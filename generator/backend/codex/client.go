package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	maximumMessageBytes  = 1 << 20
	maximumNotifications = 128
	proxyShutdownTimeout = time.Second
)

// Notification is one bounded App Server event without transport metadata.
type Notification struct {
	Method string
	Params json.RawMessage
}

// InitializeResult contains the non-secret server identity negotiated for one
// proxy connection.
type InitializeResult struct {
	UserAgent      string `json:"userAgent"`
	PlatformFamily string `json:"platformFamily"`
	PlatformOS     string `json:"platformOs"`
}

// Client is a minimal concurrent JSON-RPC 2.0 client for Codex App Server v2.
type Client struct {
	process       Process
	writeMutex    sync.Mutex
	pendingMutex  sync.Mutex
	pending       map[int64]chan rpcResponse
	nextID        atomic.Int64
	notifications chan Notification
	done          chan struct{}
	closeOnce     sync.Once
	finishOnce    sync.Once
	failureMutex  sync.Mutex
	failure       error
}

// NewClient starts reading one proxy connection. Initialize must be called
// before other App Server methods.
func NewClient(process Process) *Client {
	client := &Client{
		process:       process,
		pending:       make(map[int64]chan rpcResponse),
		notifications: make(chan Notification, maximumNotifications),
		done:          make(chan struct{}),
	}

	go client.readLoop()

	return client
}

// Initialize performs the mandatory App Server handshake and emits the
// initialized notification.
func (client *Client) Initialize(ctx context.Context) (InitializeResult, error) {
	parameters := map[string]any{
		"clientInfo": map[string]string{
			"name":    "hatmax",
			"title":   "Hatmax Generator",
			"version": AdapterVersion,
		},
		"capabilities": map[string]any{
			"experimentalApi":    false,
			"requestAttestation": false,
		},
	}

	var result InitializeResult

	err := client.Call(ctx, "initialize", parameters, &result)
	if err != nil {
		return InitializeResult{}, err
	}

	err = client.Notify("initialized", map[string]any{})
	if err != nil {
		return InitializeResult{}, err
	}

	return result, nil
}

// Call sends one correlated JSON-RPC request and decodes its result.
func (client *Client) Call(ctx context.Context, method string, parameters any, result any) error {
	if client == nil || client.process == nil {
		return backendError(eval.BackendUnavailable, method, "Codex App Server connection is unavailable")
	}

	requestID := client.nextID.Add(1)
	responseChannel := make(chan rpcResponse, 1)

	client.pendingMutex.Lock()
	client.pending[requestID] = responseChannel
	client.pendingMutex.Unlock()

	request := rpcRequest{JSONRPC: "2.0", ID: requestID, Method: method, Params: parameters}

	err := client.write(request)
	if err != nil {
		client.removePending(requestID)

		return err
	}

	select {
	case response := <-responseChannel:
		if response.Error != nil {
			return backendError(eval.BackendTurnFailed, method, boundedRPCError(response.Error))
		}

		if result == nil {
			return nil
		}

		err := json.Unmarshal(response.Result, result)
		if err != nil {
			return backendError(eval.BackendProtocolViolation, method, "Codex App Server returned an invalid result")
		}

		return nil
	case <-ctx.Done():
		client.removePending(requestID)
		_ = client.Notify("$/cancelRequest", map[string]int64{"id": requestID})

		return contextBackendError(method, ctx.Err())
	case <-client.done:
		client.removePending(requestID)

		return client.connectionError(method)
	}
}

// Notify sends one client notification without allocating a correlation ID.
func (client *Client) Notify(method string, parameters any) error {
	if client == nil || client.process == nil {
		return backendError(eval.BackendUnavailable, method, "Codex App Server connection is unavailable")
	}

	return client.write(rpcNotification{JSONRPC: "2.0", Method: method, Params: parameters})
}

// Interrupt cancels one active Codex turn through the supported v2 method.
func (client *Client) Interrupt(ctx context.Context, threadID, turnID string) error {
	parameters := map[string]string{"threadId": threadID, "turnId": turnID}

	return client.Call(ctx, "turn/interrupt", parameters, nil)
}

// Notifications exposes the bounded stream of server notifications.
func (client *Client) Notifications() <-chan Notification {
	return client.notifications
}

// Close shuts down only the owned proxy process. The resident daemon is not
// affected.
func (client *Client) Close() error {
	if client == nil || client.process == nil {
		return nil
	}

	var closeError error

	client.closeOnce.Do(func() {
		closeError = client.process.Close()

		waited := make(chan error, 1)
		go func() { waited <- client.process.Wait() }()

		select {
		case waitError := <-waited:
			closeError = errors.Join(closeError, waitError)
		case <-time.After(proxyShutdownTimeout):
			closeError = errors.Join(closeError, client.process.Kill())
		}
	})

	return closeError
}

func (client *Client) readLoop() {
	scanner := bufio.NewScanner(client.process)
	scanner.Buffer(make([]byte, 64<<10), maximumMessageBytes)

	for scanner.Scan() {
		err := client.handleMessage(scanner.Bytes())
		if err != nil {
			client.finish(err)

			return
		}
	}

	err := scanner.Err()
	if err != nil {
		client.finish(backendError(eval.BackendProtocolViolation, "read", "Codex App Server message exceeded transport bounds"))

		return
	}

	client.finish(backendError(eval.BackendUnavailable, "read", "Codex App Server proxy closed"))
}

func (client *Client) handleMessage(value []byte) error {
	var envelope rpcEnvelope

	err := json.Unmarshal(value, &envelope)
	if err != nil || envelope.JSONRPC != "2.0" {
		return backendError(eval.BackendProtocolViolation, "decode", "Codex App Server sent an invalid JSON-RPC message")
	}

	if envelope.Method != "" && len(envelope.ID) > 0 {
		_ = client.write(rpcErrorResponse{
			JSONRPC: "2.0",
			ID:      envelope.ID,
			Error: rpcError{
				Code:    -32601,
				Message: "Hatmax rejects App Server requests",
			},
		})

		return backendError(eval.BackendProtocolViolation, envelope.Method, "Codex App Server requested forbidden client activity")
	}

	if envelope.Method != "" {
		notification := Notification{Method: envelope.Method, Params: cloneRawMessage(envelope.Params)}
		select {
		case client.notifications <- notification:
			return nil
		default:
			return backendError(eval.BackendProtocolViolation, "notification", "Codex App Server notification limit exceeded")
		}
	}

	requestID, err := decodeRequestID(envelope.ID)
	if err != nil {
		return backendError(eval.BackendProtocolViolation, "response", "Codex App Server returned an invalid request ID")
	}

	client.pendingMutex.Lock()
	responseChannel, exists := client.pending[requestID]
	delete(client.pending, requestID)
	client.pendingMutex.Unlock()

	if !exists {
		return backendError(eval.BackendProtocolViolation, "response", "Codex App Server returned an unknown request ID")
	}

	responseChannel <- rpcResponse{Result: cloneRawMessage(envelope.Result), Error: envelope.Error}

	return nil
}

func (client *Client) write(value any) error {
	message, err := json.Marshal(value)
	if err != nil {
		return backendError(eval.BackendProtocolViolation, "encode", "Hatmax could not encode an App Server message")
	}

	message = append(message, '\n')

	client.writeMutex.Lock()
	defer client.writeMutex.Unlock()

	_, err = client.process.Write(message)
	if err != nil {
		return backendError(eval.BackendUnavailable, "write", "Codex App Server proxy write failed")
	}

	return nil
}

func (client *Client) removePending(requestID int64) {
	client.pendingMutex.Lock()
	delete(client.pending, requestID)
	client.pendingMutex.Unlock()
}

func (client *Client) finish(err error) {
	client.finishOnce.Do(func() {
		client.failureMutex.Lock()
		client.failure = err
		client.failureMutex.Unlock()

		close(client.done)
		close(client.notifications)
	})
}

func (client *Client) connectionError(operation string) error {
	client.failureMutex.Lock()
	defer client.failureMutex.Unlock()

	if client.failure != nil {
		return client.failure
	}

	return backendError(eval.BackendUnavailable, operation, "Codex App Server connection closed")
}

func contextBackendError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return backendError(eval.BackendTimeout, operation, "Codex App Server request timed out")
	}

	return backendError(eval.BackendCancelled, operation, "Codex App Server request was cancelled")
}

func boundedRPCError(value *rpcError) string {
	if value == nil || value.Message == "" {
		return "Codex App Server request failed"
	}

	return boundedText(value.Message, 512)
}

func decodeRequestID(value json.RawMessage) (int64, error) {
	var requestID int64

	if len(value) == 0 {
		return 0, fmt.Errorf("request ID is missing")
	}

	err := json.Unmarshal(value, &requestID)
	if err != nil || requestID < 1 {
		return 0, fmt.Errorf("request ID is invalid")
	}

	return requestID, nil
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage{}, value...)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcErrorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   rpcError        `json:"error"`
}

type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcResponse struct {
	Result json.RawMessage
	Error  *rpcError
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

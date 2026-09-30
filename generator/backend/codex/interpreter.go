// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	maximumTurnEvents = 512
	interruptTimeout  = 5 * time.Second
)

// RuntimeProvider is the resident-runtime boundary used by the interpreter.
type RuntimeProvider interface {
	Ensure(context.Context) (RuntimeInfo, error)
	OpenProxy(context.Context, RuntimeInfo) (Process, error)
}

// AppServerClient is the initialized protocol boundary used by the
// interpreter and thread manager.
type AppServerClient interface {
	Caller
	Initialize(context.Context) (InitializeResult, error)
	Notifications() <-chan Notification
	Interrupt(context.Context, string, string) error
	Failure(string) error
	Close() error
}

// ClientFactory binds an owned proxy process to an App Server client.
type ClientFactory func(Process) AppServerClient

// InterpreterConfig contains local non-secret adapter configuration.
type InterpreterConfig struct {
	ContextRoot     string
	ProjectIdentity string
	Model           string
	TurnTimeout     time.Duration
}

// Interpreter implements eval.Interpreter through Codex App Server.
type Interpreter struct {
	runtime RuntimeProvider
	factory ClientFactory
	config  InterpreterConfig
	clock   Clock
}

// NewInterpreter constructs a Codex interpreter with injectable runtime and
// protocol boundaries.
func NewInterpreter(runtime RuntimeProvider, factory ClientFactory, config InterpreterConfig, clock Clock) (*Interpreter, error) {
	if runtime == nil || factory == nil || clock == nil {
		return nil, fmt.Errorf("Codex interpreter dependencies are required")
	}

	if strings.TrimSpace(config.ContextRoot) == "" || strings.TrimSpace(config.ProjectIdentity) == "" {
		return nil, fmt.Errorf("context root and project identity are required")
	}

	if config.TurnTimeout <= 0 {
		config.TurnTimeout = DefaultTurnTimeout
	}

	return &Interpreter{runtime: runtime, factory: factory, config: config, clock: clock}, nil
}

// NewLocalInterpreter constructs the production adapter around the installed
// Codex CLI and resident App Server.
func NewLocalInterpreter(config InterpreterConfig) (*Interpreter, error) {
	return NewInterpreter(NewLocalRuntime(), func(process Process) AppServerClient {
		return NewClient(process)
	}, config, systemClock{})
}

// Interpret runs one schema-constrained Codex turn and retries once only when
// the final structured output is invalid.
func (interpreter *Interpreter) Interpret(ctx context.Context, request eval.Request) (eval.InterpreterResult, error) {
	startedAt := interpreter.clock.Now()

	runtimeInfo, err := interpreter.runtime.Ensure(ctx)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	process, err := interpreter.runtime.OpenProxy(ctx, runtimeInfo)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	client := interpreter.factory(process)
	if client == nil {
		_ = process.Close()

		return eval.InterpreterResult{}, backendError(eval.BackendUnavailable, "client_create", "Codex App Server client could not be created")
	}
	defer client.Close()

	_, err = client.Initialize(ctx)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	authenticatedIdentity, err := readAuthenticatedIdentity(ctx, client)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	identity, err := NewContextIdentity(
		interpreter.config.ProjectIdentity,
		authenticatedIdentity,
		request.Book.Version,
		request.ContractVersion,
	)
	if err != nil {
		return eval.InterpreterResult{}, backendError(eval.BackendThreadFailed, "context_identity", "Codex thread identity could not be created")
	}

	manager, err := NewThreadManager(client, interpreter.config.ContextRoot, interpreter.config.Model)
	if err != nil {
		return eval.InterpreterResult{}, backendError(eval.BackendThreadFailed, "thread_manager", "Codex thread manager could not be created")
	}

	session, err := manager.Acquire(ctx, identity)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	compiled, err := CompileTurn(session.ID, request)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	interpretation, err := interpreter.interpretTurn(ctx, client, compiled)
	if err != nil {
		return eval.InterpreterResult{}, err
	}

	return eval.InterpreterResult{
		Interpretation: interpretation,
		Provenance: eval.Provenance{
			Adapter:         "codex-app-server",
			ThreadID:        session.ID,
			ContractVersion: eval.CurrentContractVersion,
			BackendVersion:  runtimeInfo.ServerVersion,
			ProtocolVersion: ProtocolVersion,
			ModelSelection:  session.ModelSelection,
			EffectiveModel:  session.EffectiveModel,
			RuntimeReused:   runtimeInfo.Reused,
			ThreadReused:    session.Reused,
			Timing:          timingClass(interpreter.clock.Now().Sub(startedAt)),
		},
	}, nil
}

func (interpreter *Interpreter) interpretTurn(ctx context.Context, client AppServerClient, compiled CompiledTurn) (eval.Interpretation, error) {
	var lastDecodeError error

	for attempt := 0; attempt < 2; attempt++ {
		if attempt == 1 {
			compiled.Prompt = correctionPrompt(compiled.Prompt, lastDecodeError)
		}

		output, err := interpreter.runTurn(ctx, client, compiled)
		if err != nil {
			return eval.Interpretation{}, err
		}

		result, decodeErr := eval.DecodeInterpretation(output)
		if decodeErr == nil {
			return result, nil
		}

		lastDecodeError = decodeErr
	}

	message := "Codex returned invalid structured output twice"
	if lastDecodeError != nil {
		message += ": " + boundedText(lastDecodeError.Error(), 512)
	}

	return eval.Interpretation{}, backendError(eval.BackendOutputInvalid, "decode_output", message)
}

func (interpreter *Interpreter) runTurn(ctx context.Context, client AppServerClient, compiled CompiledTurn) ([]byte, error) {
	turnContext, cancel := context.WithTimeout(ctx, interpreter.config.TurnTimeout)
	defer cancel()

	var response turnStartResponse

	err := client.Call(turnContext, "turn/start", compiled.parameters(), &response)
	if err != nil {
		return nil, mapTurnError(err)
	}

	if response.Turn.ID == "" {
		return nil, backendError(eval.BackendProtocolViolation, "turn_start", "Codex App Server returned no turn identifier")
	}

	switch response.Turn.Status {
	case "completed":
		message := lastAgentMessage(response.Turn.Items)
		if strings.TrimSpace(message) == "" {
			return nil, backendError(eval.BackendOutputInvalid, "turn_start", "Codex turn returned no final agent message")
		}

		return []byte(message), nil
	case "failed":
		return nil, failedTurnError("turn_start", response.Turn)
	case "interrupted":
		return nil, backendError(eval.BackendCancelled, "turn_start", "Codex turn was interrupted")
	case "inProgress":
	default:
		return nil, backendError(eval.BackendProtocolViolation, "turn_start", "Codex turn started with an invalid status")
	}

	return waitForTurn(turnContext, client, compiled.ThreadID, response.Turn.ID, response.Turn.Items)
}

func waitForTurn(ctx context.Context, client AppServerClient, threadID, turnID string, initialItems []turnItem) ([]byte, error) {
	finalMessage := lastAgentMessage(initialItems)
	events := 0

	for {
		select {
		case notification, open := <-client.Notifications():
			if !open {
				return nil, client.Failure("turn_events")
			}

			events++
			if events > maximumTurnEvents {
				interruptTurn(client, threadID, turnID)

				return nil, backendError(eval.BackendProtocolViolation, "turn_events", "Codex turn event limit exceeded")
			}

			message, completed, err := consumeTurnNotification(notification, threadID, turnID)
			if err != nil {
				interruptTurn(client, threadID, turnID)

				return nil, err
			}

			if message != "" {
				finalMessage = message
			}

			if completed {
				if strings.TrimSpace(finalMessage) == "" {
					return nil, backendError(eval.BackendOutputInvalid, "turn_complete", "Codex turn returned no final agent message")
				}

				if len(finalMessage) > eval.MaximumInterpretationBytes {
					return nil, backendError(eval.BackendOutputInvalid, "turn_complete", "Codex final message exceeds the output limit")
				}

				return []byte(finalMessage), nil
			}
		case <-ctx.Done():
			interruptTurn(client, threadID, turnID)

			return nil, contextBackendError("turn_wait", ctx.Err())
		}
	}
}

func consumeTurnNotification(notification Notification, threadID, turnID string) (string, bool, error) {
	switch notification.Method {
	case "item/completed":
		var parameters itemCompletedParameters

		err := json.Unmarshal(notification.Params, &parameters)
		if err != nil {
			return "", false, backendError(eval.BackendProtocolViolation, "item_complete", "Codex emitted an invalid completed item")
		}

		if parameters.ThreadID != threadID || parameters.TurnID != turnID {
			return "", false, nil
		}

		if parameters.Item.Type == "agentMessage" {
			return parameters.Item.Text, false, nil
		}

		return "", false, nil
	case "turn/completed":
		var parameters turnCompletedParameters

		err := json.Unmarshal(notification.Params, &parameters)
		if err != nil {
			return "", false, backendError(eval.BackendProtocolViolation, "turn_complete", "Codex emitted an invalid completed turn")
		}

		if parameters.ThreadID != threadID || parameters.Turn.ID != turnID {
			return "", false, nil
		}

		switch parameters.Turn.Status {
		case "completed":
			return lastAgentMessage(parameters.Turn.Items), true, nil
		case "interrupted":
			return "", false, backendError(eval.BackendCancelled, "turn_complete", "Codex turn was interrupted")
		case "failed":
			return "", false, failedTurnError("turn_complete", parameters.Turn)
		default:
			return "", false, backendError(eval.BackendProtocolViolation, "turn_complete", "Codex turn completed with an invalid status")
		}
	default:
		return "", false, nil
	}
}

func readAuthenticatedIdentity(ctx context.Context, client AppServerClient) (string, error) {
	var response accountReadResponse

	err := client.Call(ctx, "account/read", accountReadParameters{RefreshToken: false}, &response)
	if err != nil {
		return "", backendError(eval.BackendAuthenticationRequired, "account_read", "Codex ChatGPT authentication could not be verified")
	}

	// RequiresOpenAIAuth describes provider policy, not whether the account is
	// signed in. A present ChatGPT account is the authentication state.
	if response.Account == nil || response.Account.Type != "chatgpt" {
		return "", backendError(eval.BackendAuthenticationRequired, "account_read", "Codex ChatGPT sign-in is required")
	}

	document, err := json.Marshal(response.Account)
	if err != nil {
		return "", backendError(eval.BackendAuthenticationRequired, "account_read", "Codex account identity could not be derived")
	}

	digest := sha256.Sum256(document)

	return hex.EncodeToString(digest[:]), nil
}

func failedTurnError(operation string, value turn) error {
	message := "Codex turn failed"
	if value.Error != nil && strings.TrimSpace(value.Error.Message) != "" {
		message += ": " + boundedText(value.Error.Message, 512)
	}

	return backendError(eval.BackendTurnFailed, operation, message)
}

func interruptTurn(client AppServerClient, threadID, turnID string) {
	ctx, cancel := context.WithTimeout(context.Background(), interruptTimeout)
	defer cancel()

	_ = client.Interrupt(ctx, threadID, turnID)
}

func correctionPrompt(original string, validationError error) string {
	diagnostic := "Hatmax rejected the previous structured result."
	if validationError != nil {
		diagnostic += " Validation diagnostic: " + boundedText(validationError.Error(), 512)
	}

	return diagnostic + " Retry once. Return only one corrected JSON value and do not use tools.\n\n" + original
}

func lastAgentMessage(items []turnItem) string {
	result := ""

	for _, item := range items {
		if item.Type == "agentMessage" {
			result = item.Text
		}
	}

	return result
}

func mapTurnError(err error) error {
	var backend eval.BackendError
	if errors.As(err, &backend) {
		if backend.Code == eval.BackendTimeout || backend.Code == eval.BackendCancelled || backend.Code == eval.BackendProtocolViolation {
			return backend
		}
	}

	return backendError(eval.BackendTurnFailed, "turn_start", "Codex turn could not be started")
}

func timingClass(duration time.Duration) eval.TimingClass {
	switch {
	case duration < time.Second:
		return eval.TimingUnderSecond
	case duration < 10*time.Second:
		return eval.TimingUnderTenSeconds
	case duration < 30*time.Second:
		return eval.TimingUnderThirtySeconds
	default:
		return eval.TimingUnderTwoMinutes
	}
}

type accountReadParameters struct {
	RefreshToken bool `json:"refreshToken"`
}

type accountReadResponse struct {
	Account            *account `json:"account"`
	RequiresOpenAIAuth bool     `json:"requiresOpenaiAuth"`
}

type account struct {
	Type     string `json:"type"`
	Email    string `json:"email,omitempty"`
	PlanType string `json:"planType,omitempty"`
}

type turnStartResponse struct {
	Turn turn `json:"turn"`
}

type turnCompletedParameters struct {
	ThreadID string `json:"threadId"`
	Turn     turn   `json:"turn"`
}

type itemCompletedParameters struct {
	ThreadID string   `json:"threadId"`
	TurnID   string   `json:"turnId"`
	Item     turnItem `json:"item"`
}

type turn struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Items  []turnItem `json:"items"`
	Error  *turnError `json:"error"`
}

type turnError struct {
	Message string `json:"message"`
}

type turnItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

var _ eval.Interpreter = (*Interpreter)(nil)

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

func TestClientInitializesAndCorrelatesRequests(t *testing.T) {
	// Initialization is followed by the required notification, and later
	// requests retain their own result correlation.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	serverDone := make(chan error, 1)

	go func() {
		decoder := json.NewDecoder(serverProcess)
		encoder := json.NewEncoder(serverProcess)

		var document map[string]json.RawMessage

		err := decoder.Decode(&document)
		if err != nil {
			serverDone <- err

			return
		}

		if _, exists := document["jsonrpc"]; exists {
			serverDone <- errors.New("initialize request contains unsupported jsonrpc field")

			return
		}

		encoded, err := json.Marshal(document)
		if err != nil {
			serverDone <- err

			return
		}

		var initialize rpcEnvelope

		err = json.Unmarshal(encoded, &initialize)
		if err != nil {
			serverDone <- err

			return
		}

		if initialize.Method != "initialize" {
			serverDone <- errors.New("initialize request missing")

			return
		}

		err = encoder.Encode(map[string]any{
			"id": json.RawMessage(initialize.ID),
			"result": map[string]string{
				"userAgent":      "codex/0.153.0",
				"platformFamily": "unix",
				"platformOs":     "linux",
			},
		})
		if err != nil {
			serverDone <- err

			return
		}

		var initialized rpcEnvelope

		err = decoder.Decode(&initialized)
		if err != nil {
			serverDone <- err

			return
		}

		if initialized.Method != "initialized" || len(initialized.ID) != 0 {
			serverDone <- errors.New("initialized notification missing")

			return
		}

		serverDone <- nil
	}()

	result, err := client.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if result.UserAgent != "codex/0.153.0" || result.PlatformOS != "linux" {
		t.Fatalf("Initialize() result = %#v", result)
	}

	err = <-serverDone
	if err != nil {
		t.Fatalf("server error = %v", err)
	}
}

func TestClientCorrelatesConcurrentResponses(t *testing.T) {
	// Responses may arrive out of order without crossing request results.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	go func() {
		decoder := json.NewDecoder(serverProcess)
		encoder := json.NewEncoder(serverProcess)
		requests := make([]rpcEnvelope, 2)
		_ = decoder.Decode(&requests[0])
		_ = decoder.Decode(&requests[1])
		_ = encoder.Encode(map[string]any{"id": json.RawMessage(requests[1].ID), "result": map[string]string{"value": requests[1].Method}})
		_ = encoder.Encode(map[string]any{"id": json.RawMessage(requests[0].ID), "result": map[string]string{"value": requests[0].Method}})
	}()

	type result struct {
		Value string `json:"value"`
	}

	var first result

	var second result

	var wait sync.WaitGroup

	wait.Add(2)

	go func() {
		defer wait.Done()

		err := client.Call(context.Background(), "first", map[string]any{}, &first)
		if err != nil {
			t.Errorf("first Call() error = %v", err)
		}
	}()

	go func() {
		defer wait.Done()

		err := client.Call(context.Background(), "second", map[string]any{}, &second)
		if err != nil {
			t.Errorf("second Call() error = %v", err)
		}
	}()

	wait.Wait()

	if first.Value != "first" || second.Value != "second" {
		t.Fatalf("results = %#v, %#v", first, second)
	}
}

func TestClientCancellationSendsCancelNotification(t *testing.T) {
	// Context cancellation is stable and asks the server to discard the request.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	observed := make(chan rpcEnvelope, 2)

	go func() {
		decoder := json.NewDecoder(serverProcess)

		for index := 0; index < 2; index++ {
			var message rpcEnvelope

			_ = decoder.Decode(&message)
			observed <- message
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Call(ctx, "thread/list", map[string]any{}, nil)
	assertBackendCode(t, err, eval.BackendCancelled)

	request := <-observed
	cancellation := <-observed

	if request.Method != "thread/list" || cancellation.Method != "$/cancelRequest" {
		t.Fatalf("methods = %q, %q", request.Method, cancellation.Method)
	}
}

func TestClientRejectsServerRequests(t *testing.T) {
	// Any server-to-client request is denied and closes the unsafe connection.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	encoder := json.NewEncoder(serverProcess)

	err := encoder.Encode(map[string]any{
		"id":     44,
		"method": "item/tool/call",
		"params": map[string]any{},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoder := json.NewDecoder(serverProcess)

	var response rpcEnvelope

	err = decoder.Decode(&response)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if response.Error == nil || response.Error.Code != -32601 {
		t.Fatalf("response = %#v", response)
	}

	select {
	case <-client.done:
		assertBackendCode(t, client.connectionError("test"), eval.BackendProtocolViolation)
	case <-time.After(time.Second):
		t.Fatal("client did not reject server request")
	}
}

func TestClientInterruptsTurns(t *testing.T) {
	// Turn interruption uses the exact v2 method and identifiers.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	go func() {
		decoder := json.NewDecoder(serverProcess)
		encoder := json.NewEncoder(serverProcess)

		var request rpcEnvelope

		_ = decoder.Decode(&request)
		_ = encoder.Encode(map[string]any{"id": json.RawMessage(request.ID), "result": map[string]any{}})
	}()

	err := client.Interrupt(context.Background(), "thread-1", "turn-1")
	if err != nil {
		t.Fatalf("Interrupt() error = %v", err)
	}
}

func TestClientBoundsNotifications(t *testing.T) {
	// An unread notification flood closes the connection at the fixed limit.
	clientProcess, serverProcess := newPipeProcesses()
	client := NewClient(clientProcess)

	t.Cleanup(func() { _ = client.Close() })

	writer := bufio.NewWriter(serverProcess)
	for index := 0; index <= maximumNotifications; index++ {
		_, _ = writer.WriteString(`{"method":"thread/status/changed","params":{}}` + "\n")
	}

	_ = writer.Flush()

	select {
	case <-client.done:
		assertBackendCode(t, client.connectionError("test"), eval.BackendProtocolViolation)
	case <-time.After(time.Second):
		t.Fatal("client did not bound notifications")
	}
}

type pipeProcess struct {
	net.Conn
	waitOnce sync.Once
	wait     chan struct{}
}

func newPipeProcesses() (*pipeProcess, *pipeProcess) {
	clientConnection, serverConnection := net.Pipe()

	return newPipeProcess(clientConnection), newPipeProcess(serverConnection)
}

func newPipeProcess(connection net.Conn) *pipeProcess {
	return &pipeProcess{Conn: connection, wait: make(chan struct{})}
}

func (process *pipeProcess) Close() error {
	process.waitOnce.Do(func() { close(process.wait) })

	return process.Conn.Close()
}

func (process *pipeProcess) Wait() error {
	<-process.wait

	return nil
}

func (process *pipeProcess) Kill() error { return process.Close() }

var _ io.ReadWriteCloser = (*pipeProcess)(nil)

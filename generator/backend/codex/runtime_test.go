// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package codex

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

func TestRuntimeReusesCompatibleDaemon(t *testing.T) {
	// A compatible daemon must be reused without any lifecycle mutation.
	runner := &fakeRunner{outputs: []fakeOutput{
		{value: []byte("codex-cli 0.153.0\n")},
		{value: []byte(`{"cli_version":"0.153.0","server_version":"0.153.2"}`)},
	}}
	runtime := NewRuntime(fakeLocator{path: "/usr/bin/codex"}, runner, fakeClock{})

	result, err := runtime.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if !result.Reused || result.CLIVersion != "0.153.0" || result.ServerVersion != "0.153.2" {
		t.Fatalf("Ensure() result = %#v", result)
	}

	want := [][]string{{"--version"}, {"app-server", "daemon", "version"}}
	if !reflect.DeepEqual(runner.arguments, want) {
		t.Fatalf("commands = %#v, want %#v", runner.arguments, want)
	}
}

func TestRuntimeStartsAbsentDaemonOnce(t *testing.T) {
	// Absence admits start, but never restart or stop.
	runner := &fakeRunner{outputs: []fakeOutput{
		{value: []byte("codex-cli 0.153.0\n")},
		{err: errors.New("failed to connect: no such file")},
		{value: []byte("started\n")},
		{value: []byte(`{"serverVersion":"0.153.0"}`)},
	}}
	runtime := NewRuntime(fakeLocator{path: "/usr/bin/codex"}, runner, fakeClock{})

	result, err := runtime.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if result.Reused {
		t.Fatal("Ensure() reported an absent daemon as reused")
	}

	want := [][]string{
		{"--version"},
		{"app-server", "daemon", "version"},
		{"app-server", "daemon", "start"},
		{"app-server", "daemon", "version"},
	}
	if !reflect.DeepEqual(runner.arguments, want) {
		t.Fatalf("commands = %#v, want %#v", runner.arguments, want)
	}
}

func TestRuntimeRejectsIncompatibleDaemonWithoutMutation(t *testing.T) {
	// A running incompatible daemon requires user action and is not replaced.
	runner := &fakeRunner{outputs: []fakeOutput{
		{value: []byte("codex-cli 0.153.0\n")},
		{value: []byte(`{"server_version":"0.152.0"}`)},
	}}
	runtime := NewRuntime(fakeLocator{path: "/usr/bin/codex"}, runner, fakeClock{})

	_, err := runtime.Ensure(context.Background())
	assertBackendCode(t, err, eval.BackendIncompatible)

	if len(runner.arguments) != 2 {
		t.Fatalf("command count = %d, want 2", len(runner.arguments))
	}
}

func TestRuntimeReportsMissingExecutable(t *testing.T) {
	// Missing Codex is distinct from daemon startup and protocol failures.
	runtime := NewRuntime(fakeLocator{err: errors.New("not found")}, &fakeRunner{}, fakeClock{})

	_, err := runtime.Ensure(context.Background())
	assertBackendCode(t, err, eval.BackendUnavailable)
}

func TestRuntimeOpensOwnedProxy(t *testing.T) {
	// The proxy is a child connection; daemon lifecycle commands are excluded.
	process, server := newPipeProcesses()
	runner := &fakeRunner{process: process}
	runtime := NewRuntime(fakeLocator{}, runner, fakeClock{})
	serverDone := serveTestWebSocket(server)

	got, err := runtime.OpenProxy(context.Background(), RuntimeInfo{Executable: "/usr/bin/codex"})
	if err != nil {
		t.Fatalf("OpenProxy() error = %v", err)
	}

	if _, ok := got.(*webSocketProcess); !ok {
		t.Fatalf("OpenProxy() process = %T, want WebSocket transport", got)
	}

	want := [][]string{{"app-server", "proxy"}}
	if !reflect.DeepEqual(runner.started, want) {
		t.Fatalf("started = %#v, want %#v", runner.started, want)
	}

	err = <-serverDone
	if err != nil {
		t.Fatalf("WebSocket server error = %v", err)
	}

	_ = got.Close()
}

func assertBackendCode(t *testing.T, err error, want eval.BackendFailureCode) {
	t.Helper()

	var backend eval.BackendError
	if !errors.As(err, &backend) {
		t.Fatalf("error = %v, want eval.BackendError", err)
	}

	if backend.Code != want {
		t.Fatalf("code = %q, want %q", backend.Code, want)
	}
}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(42, 0) }

type fakeLocator struct {
	path string
	err  error
}

func (locator fakeLocator) LookPath(string) (string, error) { return locator.path, locator.err }

type fakeOutput struct {
	value []byte
	err   error
}

type fakeRunner struct {
	outputs   []fakeOutput
	arguments [][]string
	started   [][]string
	process   Process
}

func (runner *fakeRunner) Output(_ context.Context, _ string, arguments ...string) ([]byte, error) {
	runner.arguments = append(runner.arguments, append([]string{}, arguments...))
	if len(runner.outputs) == 0 {
		return nil, errors.New("unexpected command")
	}

	output := runner.outputs[0]
	runner.outputs = runner.outputs[1:]

	return output.value, output.err
}

func (runner *fakeRunner) Start(_ context.Context, _ string, arguments ...string) (Process, error) {
	runner.started = append(runner.started, append([]string{}, arguments...))

	return runner.process, nil
}

type fakeProcess struct{}

func (*fakeProcess) Read([]byte) (int, error)        { return 0, io.EOF }
func (*fakeProcess) Write(value []byte) (int, error) { return len(value), nil }
func (*fakeProcess) Close() error                    { return nil }
func (*fakeProcess) Wait() error                     { return nil }
func (*fakeProcess) Kill() error                     { return nil }

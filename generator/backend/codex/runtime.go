// Package codex connects the Hatmax interpreter boundary to a local Codex App
// Server without granting the server authority over planning or execution.
package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	// ProtocolVersion is the Codex App Server protocol implemented by this
	// adapter.
	ProtocolVersion = "v2"
	// AdapterVersion identifies the Hatmax-owned Codex adapter contract.
	AdapterVersion = "1"
)

var semanticVersionPattern = regexp.MustCompile(`\b(\d+)\.(\d+)\.(\d+)(?:[-+][0-9A-Za-z.-]+)?\b`)

// Clock supplies observation time without coupling runtime tests to the wall
// clock.
type Clock interface {
	Now() time.Time
}

// Locator resolves an executable without invoking it.
type Locator interface {
	LookPath(string) (string, error)
}

// CommandRunner executes bounded runtime-management commands.
type CommandRunner interface {
	Output(context.Context, string, ...string) ([]byte, error)
	Start(context.Context, string, ...string) (Process, error)
}

// Process is one short-lived child process owned by a Hatmax client.
type Process interface {
	io.ReadWriteCloser
	Wait() error
	Kill() error
}

// RuntimeInfo records a compatible resident App Server observation.
type RuntimeInfo struct {
	Executable    string
	CLIVersion    string
	ServerVersion string
	Reused        bool
	CheckedAt     time.Time
}

// Runtime discovers and manages the local Codex App Server lifecycle.
type Runtime struct {
	locator Locator
	runner  CommandRunner
	clock   Clock
}

// NewRuntime constructs a runtime with injectable operating-system
// boundaries.
func NewRuntime(locator Locator, runner CommandRunner, clock Clock) *Runtime {
	return &Runtime{locator: locator, runner: runner, clock: clock}
}

// NewLocalRuntime constructs the production local runtime.
func NewLocalRuntime() *Runtime {
	return NewRuntime(systemLocator{}, systemRunner{}, systemClock{})
}

// Ensure verifies the installed CLI and resident App Server, starting the
// managed daemon only when it is absent. It never stops or restarts a daemon.
func (runtime *Runtime) Ensure(ctx context.Context) (RuntimeInfo, error) {
	if runtime == nil || runtime.locator == nil || runtime.runner == nil || runtime.clock == nil {
		return RuntimeInfo{}, backendError(eval.BackendUnavailable, "runtime_check", "Codex runtime dependencies are unavailable")
	}

	executable, err := runtime.locator.LookPath("codex")
	if err != nil {
		return RuntimeInfo{}, backendError(eval.BackendUnavailable, "runtime_discovery", "Codex executable was not found")
	}

	cliOutput, err := runtime.runner.Output(ctx, executable, "--version")
	if err != nil {
		return RuntimeInfo{}, backendError(eval.BackendUnavailable, "cli_version", "Codex CLI version could not be read")
	}

	cliVersion, err := parseSemanticVersion(string(cliOutput))
	if err != nil {
		return RuntimeInfo{}, backendError(eval.BackendIncompatible, "cli_version", "Codex CLI returned an unsupported version")
	}

	serverOutput, versionErr := runtime.runner.Output(ctx, executable, "app-server", "daemon", "version")

	reused := versionErr == nil
	if versionErr != nil {
		if !daemonAbsent(versionErr) {
			return RuntimeInfo{}, backendError(eval.BackendIncompatible, "server_version", "running Codex App Server compatibility could not be established")
		}

		_, startErr := runtime.runner.Output(ctx, executable, "app-server", "daemon", "start")
		if startErr != nil {
			return RuntimeInfo{}, backendError(eval.BackendStartFailed, "daemon_start", "Codex App Server daemon could not be started")
		}

		serverOutput, versionErr = runtime.runner.Output(ctx, executable, "app-server", "daemon", "version")
		if versionErr != nil {
			return RuntimeInfo{}, backendError(eval.BackendStartFailed, "server_version", "started Codex App Server did not become available")
		}
	}

	serverVersion, err := parseDaemonVersion(serverOutput)
	if err != nil || !compatibleVersions(cliVersion, serverVersion) {
		return RuntimeInfo{}, backendError(eval.BackendIncompatible, "server_version", "Codex CLI and App Server versions are incompatible")
	}

	return RuntimeInfo{
		Executable:    executable,
		CLIVersion:    cliVersion,
		ServerVersion: serverVersion,
		Reused:        reused,
		CheckedAt:     runtime.clock.Now(),
	}, nil
}

// OpenProxy starts one short-lived stdio proxy owned by the caller. Closing
// the returned process cannot stop the resident daemon.
func (runtime *Runtime) OpenProxy(ctx context.Context, info RuntimeInfo) (Process, error) {
	if runtime == nil || runtime.runner == nil || info.Executable == "" {
		return nil, backendError(eval.BackendUnavailable, "proxy_start", "Codex runtime is unavailable")
	}

	process, err := runtime.runner.Start(ctx, info.Executable, "app-server", "proxy")
	if err != nil {
		return nil, backendError(eval.BackendStartFailed, "proxy_start", "Codex App Server proxy could not be started")
	}

	websocket, err := openWebSocketProcess(ctx, process)
	if err != nil {
		_ = process.Kill()
		_ = process.Close()

		return nil, err
	}

	return websocket, nil
}

func parseSemanticVersion(value string) (string, error) {
	match := semanticVersionPattern.FindStringSubmatch(value)
	if len(match) == 0 {
		return "", errors.New("semantic version not found")
	}

	return match[0], nil
}

func parseDaemonVersion(value []byte) (string, error) {
	var document map[string]any
	if json.Unmarshal(value, &document) == nil {
		for _, key := range []string{"server_version", "serverVersion", "app_server_version", "appServerVersion", "daemon_version", "daemonVersion"} {
			if candidate, ok := document[key].(string); ok {
				return parseSemanticVersion(candidate)
			}
		}
	}

	return parseSemanticVersion(string(value))
}

func compatibleVersions(cliVersion, serverVersion string) bool {
	cli := semanticVersionPattern.FindStringSubmatch(cliVersion)
	server := semanticVersionPattern.FindStringSubmatch(serverVersion)

	return len(cli) > 2 && len(server) > 2 && cli[1] == server[1] && cli[2] == server[2]
}

func daemonAbsent(err error) bool {
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "failed to connect") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "no such file")
}

func backendError(code eval.BackendFailureCode, operation, message string) error {
	return eval.BackendError{Code: code, Operation: operation, Message: message}
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type systemLocator struct{}

func (systemLocator) LookPath(file string) (string, error) { return exec.LookPath(file) }

type systemRunner struct{}

func (systemRunner) Output(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, arguments...)

	var stderr bytes.Buffer

	command.Stderr = &stderr

	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w: %s", err, boundedText(stderr.String(), 512))
	}

	return output, nil
}

func (systemRunner) Start(ctx context.Context, name string, arguments ...string) (Process, error) {
	command := exec.CommandContext(ctx, name, arguments...)

	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin: %w", err)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()

		return nil, fmt.Errorf("open stdout: %w", err)
	}

	err = command.Start()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()

		return nil, fmt.Errorf("start proxy: %w", err)
	}

	return &commandProcess{command: command, stdin: stdin, stdout: stdout}, nil
}

type commandProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
}

func (process *commandProcess) Read(value []byte) (int, error)  { return process.stdout.Read(value) }
func (process *commandProcess) Write(value []byte) (int, error) { return process.stdin.Write(value) }

func (process *commandProcess) Close() error {
	return errors.Join(process.stdin.Close(), process.stdout.Close())
}

func (process *commandProcess) Wait() error { return process.command.Wait() }

func (process *commandProcess) Kill() error {
	if process.command.Process == nil {
		return nil
	}

	return process.command.Process.Kill()
}

func boundedText(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}

	return value[:maximum]
}

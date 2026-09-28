# Slice 2: Codex App Server Runtime

Status: reviewing
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Branch: `feat/generator-codex-app-server`
PR: #33

## Purpose

Provide a bounded, reusable local connection to the installed Codex App
Server without granting it authority over Hatmax interpretation, planning, or
project execution.

## Delivered Behavior

The Codex runtime discovers the installed executable, reads the CLI and daemon
versions, reuses a compatible resident daemon, and starts the managed daemon
only when the control socket is absent. It never invokes daemon restart or
stop. Each Hatmax client owns only its short-lived stdio proxy.

The App Server client implements the required JSON-RPC 2.0 initialization,
correlated concurrent requests, bounded notifications and messages, context
cancellation, `turn/interrupt`, and owned proxy shutdown. Server-to-client
requests are denied and close the connection with a stable protocol failure.

Thread management derives an opaque identity from the project, Book, adapter
contract, and authenticated Codex identity. It starts or resumes a thread in a
Hatmax-owned `0700` context directory that contains no target-project files or
path. Threads use read-only sandboxing, no approval channel, no dynamic MCP
configuration, disabled tool-related features, and backend-default model
selection unless an explicit model is configured. Any observed tool activity
closes the connection.

## Contracts Added or Changed

- `Runtime.Ensure` owns executable discovery, compatibility checks, and
  start-if-absent daemon behavior.
- `Runtime.OpenProxy` owns one client proxy without owning the resident daemon.
- `Client` implements the minimal App Server v2 JSON-RPC transport boundary.
- `NewContextIdentity` creates opaque contract-bound project identities.
- `ThreadManager.Acquire` lists, resumes, or starts isolated project threads.
- transport guards reject server requests, tool notifications, executable
  item kinds, oversized messages, and notification floods.

## Files of Interest

- `generator/backend/codex/runtime.go`
- `generator/backend/codex/runtime_test.go`
- `generator/backend/codex/client.go`
- `generator/backend/codex/client_test.go`
- `generator/backend/codex/thread.go`
- `generator/backend/codex/thread_test.go`

## Validation

- `go test ./generator/backend/codex/...` passed.
- `go test -race ./generator/backend/codex/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

This slice validates runtime and protocol behavior with injected process
boundaries and deterministic fake transports. It does not start the user's
daemon or perform an authenticated model turn. Slice 3 must run the first live
compatibility smoke against the installed Codex CLI and App Server, validate
the current thread configuration fields, and exercise the structured-output
schema through a real interpretation turn.

The shared daemon remains a logged-in user process and is not an operating-
system confidentiality boundary. Hatmax instead withholds the target path and
contents, uses an isolated working directory, configures read-only and
tool-free behavior, and fails closed if tool activity appears.

# Slice 5: Terminal Product Surface

Status: reviewing
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Branch: `feat/generator-terminal-surface`
PR: `#36` (open)

## Purpose

Expose the approved interactive lifecycle as the first usable
`hatmax generate` command and prove its terminal-facing product boundary.

## Delivered Behavior

The thin `cmd/hatmax` entrypoint binds process cancellation to a testable
line-oriented application under `internal/hatmaxcli`. The command accepts
exactly `hatmax generate "<request>"`, treats the current directory as the
target project root, obtains focused clarification answers, displays the
canonical plan before mutation, and requires an explicit `y` or `yes`
approval. Rejection or unavailable input cancels without executing the plan.

Production assembly connects the terminal ports to the resident Codex
interpreter and backend-neutral interaction coordinator. Project identity is a
canonical-path digest used only for isolated Codex context; the target path is
not added to interpreter context. The command maps completed, cancelled,
clarification-required, unsupported, rejected, stale, execution-failed, and
unclassified outcomes to stable process exit statuses.

Terminal results report the plan identity and affected surfaces, bounded
interpreter provenance for each turn, pending clarification, project drift,
execution changes, changed and unchanged surfaces, independent conformance,
exact repository-command arguments and exit evidence, repairs, warnings, and
stable diagnostics. Reporting does not expose authentication data, reasoning,
raw App Server events, or automatic approval.

The terminal acceptance suite drives the real coordinator with a fake
interpreter through `create_feature`, `add_field`, and `add_validation`. It
also covers approval rejection, unsupported intent, unavailable backend,
approval-time drift, and post-commit repository validation failure. Each
successful operation verifies that observed project changes are declared by
the deterministic execution manifest, and the suite verifies that Hatmax
creates no Git state.

The completed user-visible behavior is recorded once in `CHANGELOG.md` under
`Unreleased`.

## Contracts Added or Changed

- `hatmax generate "<request>"` is the initial public command surface.
- `hatmaxcli.Config` supplies terminal IO, working-directory discovery, and
  coordinator assembly for deterministic testing.
- `interaction.Result.Manifest` exposes the deterministic execution contract
  needed to correlate applied edits with changed and unchanged surfaces.
- Exit statuses distinguish usage, cancellation, unresolved clarification,
  unsupported requests, rejected intent, stale plans, execution failure, and
  success.

## Files of Interest

- `cmd/hatmax/main.go`
- `internal/hatmaxcli/app.go`
- `internal/hatmaxcli/local.go`
- `internal/hatmaxcli/report.go`
- `internal/hatmaxcli/acceptance_test.go`
- `generator/interaction/types.go`
- `generator/interaction/execute.go`
- `CHANGELOG.md`

## Validation

- `go test ./cmd/hatmax/... ./internal/hatmaxcli/... ./generator/interaction/...`
  passed.
- `go test -race ./generator/interaction/... ./generator/backend/codex/...`
  passed.
- `go test ./generator/...` passed.
- `newgrp docker` followed by `make check` passed with 84.0% statement
  coverage and zero strict-lint issues.
- `make docs-check` passed.
- `git diff --check` passed.
- `make generator-live-smoke` ran against the slice candidate and stopped
  before inference with `backend_start_failed during daemon_start`.

Direct inspection confirmed the smoke prerequisite: `codex-cli 0.153.0` is
installed through mise, but `codex app-server daemon start` requires the
managed standalone executable at
`~/.codex/packages/standalone/current/codex`, which is absent. No credentials,
model output, target-project content, or inference result were exposed.

## Risks and Follow-ups

The deterministic terminal surface and full repository gate are green. The
authenticated Codex behavior is not yet proven for this candidate because the
host cannot start the resident App Server daemon. Installing the standalone
Codex distribution is an explicit machine-level prerequisite and was not
performed as part of this slice.

After merge, the delivery set remains open until the exact integrated `dev`
candidate passes the authenticated smoke, demonstrates same-project daemon
and thread reuse, and demonstrates different-project thread isolation. No
`main` alignment, mirror update, tag, or release is implied.

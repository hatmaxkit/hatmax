<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 2: Atomic Application

Status: delivered
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Branch: `feat/generator-atomic-application`
PR: `#28` (merged)

## Purpose

Apply a prepared manifest as one recoverable filesystem transaction with no
partial success.

## Delivered Behavior

The execution package now opens a bounded workspace from one sealed manifest,
its sealed plan, and a compatible project inventory. Opening the workspace
re-inspects the project, verifies manifest and plan identity, revalidates the
project fingerprint, rejects unsupported targets, and snapshots every admitted
file before any renderer output is accepted.

Renderers stage content by manifest edit ID rather than filesystem path.
Structured mutations support complete file creation, complete replacement, and
insertion immediately before or after one exact anchor. Staging is in-memory,
copies caller-owned bytes, enforces per-file and per-workspace bounds, and
checks declared postconditions including Go parsing and required content.

Commit revalidates the relevant project fingerprint and every target snapshot
before writing. Changed targets are prepared as same-directory temporary files,
persisted, permissioned, and renamed in manifest order. Each target is checked
again immediately before replacement and verified after replacement.

If application fails, committed targets are restored in reverse order from the
captured rollback material. Rollback restores original content and mode or
removes newly created files, cleans transaction-owned temporary files and empty
directories, and refuses to overwrite a concurrent external change. Stable
diagnostics distinguish drift, target conflict, application failure, successful
rollback, and rollback conflict with manual recovery paths.

Committing an already committed workspace verifies every desired target and
returns `already_satisfied` without writing. Structured insertion also reports
`already_satisfied` when content is already adjacent to its exact anchor and
returns a semantic conflict when the anchor is missing, ambiguous, or the
content exists elsewhere.

## Implementation Notes

The workspace permits at most 1,024 manifest edits, 4 MiB per original or
staged file, and 64 MiB of staged content. Paths remain project-relative and
parent components may not be symbolic links.

Atomicity is recoverable across multiple files: each individual replacement is
an atomic same-filesystem rename, while the workspace owns deterministic
reverse-order rollback for a later failure. External database, network,
deployment, and publication effects remain outside this transaction.

No repository command is executed by this slice. Command execution and its
evidence remain assigned to Slice 5 after conformance succeeds.

## Contracts Added or Changed

The public execution contract adds `Workspace`, `Mutation`, `Change`, `Result`,
and their status and position enums, together with `OpenWorkspace`, `Stage`,
`Changes`, and `Commit`.

## Files of Interest

- `generator/execute/workspace.go`
- `generator/execute/stage.go`
- `generator/execute/commit.go`
- `generator/execute/workspace_test.go`
- `generator/execute/commit_test.go`
- `generator/execute/atomic_behavior_test.go`

## Validation

- `go test ./generator/execute/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Execution package tests report 78.7% statement coverage.

## Risks and Follow-ups

This slice accepts already rendered content but does not define Book-owned
render recipes. Slice 3 owns complete canonical CRUD rendering and wiring.
Slice 4 owns semantic discovery for incremental field and validation mutation.
Slice 5 owns independent conformance and bounded repository-command execution.

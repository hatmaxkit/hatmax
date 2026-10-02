<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 8: Builder Acceptance

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `test/conversational-builder-acceptance`
PR: #63

## Purpose

Prove the conversational Hatmax builder across deterministic workflow,
generated-project, persistence, terminal, and authenticated Codex boundaries,
then document the product surface that users can operate.

## Delivered Behavior

The acceptance suite exercises application creation with an initial feature,
plan cancellation and revision, approval, project rebinding, close and reopen,
later feature evolution, reset, source drift, infrastructure-blocked
validation, and off-domain rejection. It asserts that only approved operations
invoke the interaction kernel and that completed application creation moves
the conversation into exact project scope.

The authenticated Codex smoke now covers ordinary dialogue, existing-project
intent, documentation intent, focused application clarification, a complete
application plan, approval recomputation, runtime and thread reuse, scope
rebinding, and independent project isolation. The smoke found and corrected a
prompt rule that discarded an explicitly supplied application module path
during provisional target inspection. Live smoke execution disables Go test
caching so every claimed observation performs new inference.

The public documentation now presents `hm` as the canonical conversational
builder. The User Guide follows one application from a parent directory
through scaffold approval and subsequent project evolution. The reference
records commands, controls, supported operations, state and privacy,
resident-runtime behavior, validation outcomes, exit statuses, and the
temporary `hatmax` compatibility policy. The README and Unreleased changelog
use the same product language.

## Implementation Notes

The deterministic workflow acceptance uses an instrumented conversation
engine around the real state machine and local store. It observes orchestration
without granting fixtures an alternate mutation path. The authenticated
application approval uses the real interpreter and planning kernel, verifies
that recomputation preserves the displayed digest, and limits materialization
to the minimum project identity needed to prove post-completion rebinding.

The live environment exposed three Codex versions. The interactive `PATH`
selected mise Codex `0.153.0`, a separate local binary was `0.158.0`, and the
resident managed App Server was `0.159.0`. Validation used the managed
`0.159.0` executable explicitly, matching the resident server without stopping
or restarting the user's session.

## Files of Interest

- `generator/conversation/builder_acceptance_test.go`
- `generator/backend/codex/live_smoke_test.go`
- `generator/backend/codex/prompt.go`
- `docs/reference/generator/README.md`
- `docs/tutorials/user-guide/assisted-generation.md`
- `README.md`
- `CHANGELOG.md`
- `Makefile`

## Validation

- `make generator-scaffold-acceptance` passed.
- `make generator-conversation-acceptance` passed.
- `go test -race ./generator/... ./internal/hatmaxtui/... ./internal/hatmaxcli/...` passed.
- `PATH=<managed-codex-0.159.0>:$PATH make generator-live-smoke` passed with a
  non-cached authenticated run in 88.79 seconds.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `make docs-check` passed.
- `git diff --check` passed.
- `make check` did not complete because Testcontainers could not access
  `/var/run/docker.sock`; Docker returned permission denied before Postgres
  integration containers started. Generator, TUI, race, live Codex, vet, lint,
  and documentation gates are independently green.

Post-merge validation on exact integrated `dev` commit
`c854355a7b4d4effa767523a87e5827b28c80eb0` repeated the scaffold,
conversation, race, vet, lint, documentation, and whitespace gates
successfully. The non-cached authenticated smoke passed again in 86.58 seconds.

## Remaining Delivery Gate

The slice is delivered by #63. The Docker-dependent portion of the integrated
delivery-set gate remains unverified until the runner or local user can access
the Docker socket; it is not represented as passed.

<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Conversational Hatmax Builder Delivery Plan

Status: Delivered
Delivery set: conversational-hatmax-builder
Slice strategy: layered
Reason: application bootstrap and the conversational TUI cross existing Book,
intent, planning, execution, Codex, persistence, coordination, and terminal
boundaries. Contracts and deterministic kernels must precede product
integration so neither the TUI nor Codex becomes an alternate mutation
authority.
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`
Approved specs:

- `ops/default/spec/canonical-application-scaffold.md`
- `ops/default/spec/conversational-hatmax-surface.md`
- `ops/default/spec/conversational-hatmax-surface-model.md`

Tracker: `ops/default/tracker/conversational-hatmax-builder.md`
Base branch: `dev`
Planning base: `510a856419a97d64959387099c48a81f867fb09a`
Active slice: none; all eight slices are delivered
Execution gate: satisfied by `cbea3e0838c4ef25229b3822aebcf0f1dbcd2a91`

## Objective

Deliver Hatmax as a conversational application builder. A user can start `hm`
from a parent directory, create a compiling canonical Hatmax application, add
optional initial features through one visible plan, and later resume a
project-scoped conversation to evolve the application.

Conversation remains broad and informal. Only typed operations admitted by the
Hatmax Book become project mutations. Codex remains a tool-less interpreter;
Hatmax owns inspection, completeness, planning, approval, rendering, mutation,
conformance, and validation.

## Delivered Prerequisites

- The Book, intent, planning, execution, and conformance sets provide rules,
  sealed plans, atomic edits, and stable diagnostics.
- The product-surface set provides the resident Codex App Server, isolated
  threads, bounded interpretation, coordination, approval, and headless CLI.
- The boxed-documentation set preserves explicit-only Diataxis generation.
- The curated quiz records accepted scaffold and conversational behavior.

## Planning Decisions

- Slice 1 resolves remaining draft questions and promotes all affected specs
  together. No runtime work starts from conflicting drafts.
- `create_application` is a first-class typed operation. The session directory
  is the default parent and the project slug selects a child target.
- A scaffold must compile and conform. Tests blocked by unavailable declared
  infrastructure are non-blocking; observed generated-test failures block.
- `main.go` contains only `main`; `internal/application` owns construction and
  lifecycle assembly.
- Initial features are visible dependent units in one sealed application plan.
  Later single-feature work retains a compact interaction.
- `hm` is canonical for TUI and headless use. `hatmax` remains a temporary
  compatibility alias whose retirement policy is resolved in Slice 1.
- Conversation state is local user state, never repository source or reusable
  approval.
- The TUI reuses the delivered resident Codex daemon and project isolation.
- Plan presentation is provisionally compact for routine work and expanded for
  composite bootstrap; Slice 7 validates that choice.

## Scope

The delivery includes promoted and reconciled contracts; pre-project inventory
and drift; application identity and module-path normalization; deterministic
scaffold and composite planning; canonical rendering, staging, publication,
compilation, and conformance; adaptive conversation; local conversation
persistence; resume, reset, cancellation, and recovery; the `hm` TUI and
headless parity; deterministic and authenticated acceptance; and final product
documentation and changelog updates.

It excludes unrestricted coding-agent behavior, non-Hatmax project generation,
automatic approval, arbitrary model tools, Git publication, deployment, live
database migration, and unadmitted capabilities.

## Architecture Boundaries

```text
generator/book/             Application archetype and rules
generator/project/          Parent, target, and project inventory
generator/intent/           Application and mutation intents
generator/plan/             Scaffold and composite plans
generator/eval/             Dialogue and interpretation contracts
generator/backend/codex/    Tool-less resident backend
generator/execute/          Rendering, staging, publication, conformance
generator/conversation/     Conversation state machine and store contracts
generator/interaction/      End-to-end coordination
internal/hatmaxstate/       User-local persistence adapter
internal/hatmaxtui/         Terminal presentation and input
internal/hatmaxcli/         Headless presentation and compatibility
cmd/hm/                     Canonical executable
cmd/hatmax/                 Temporary compatibility entrypoint
```

Slice 1 may revise package names before spec promotion, but it must preserve
ownership. TUI code cannot plan or mutate. Codex cannot inspect files, select
obligations, or execute. Conversation state cannot replace source truth.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Builder contracts | `docs/hatmax-builder-contracts` | `docs(slice-1): define Hatmax builder contracts` | `ops/default/report/slices/conversational-hatmax-builder/slice-1-builder-contracts.md` |
| Slice 2 | Application planning kernel | `feat/application-scaffold-kernel` | `feat(slice-2): plan canonical Hatmax applications` | `ops/default/report/slices/conversational-hatmax-builder/slice-2-application-planning-kernel.md` |
| Slice 3 | Scaffold execution | `feat/application-scaffold-rendering` | `feat(slice-3): render canonical Hatmax applications` | `ops/default/report/slices/conversational-hatmax-builder/slice-3-scaffold-execution.md` |
| Slice 4 | Application bootstrap product | `feat/application-scaffold-product` | `feat(slice-4): deliver Hatmax application bootstrap` | `ops/default/report/slices/conversational-hatmax-builder/slice-4-application-bootstrap-product.md` |
| Slice 5 | Conversation state | `feat/conversational-state` | `feat(slice-5): persist Hatmax conversations` | `ops/default/report/slices/conversational-hatmax-builder/slice-5-conversation-state.md` |
| Slice 6 | Conversational coordinator | `feat/conversational-coordinator` | `feat(slice-6): coordinate conversational Hatmax work` | `ops/default/report/slices/conversational-hatmax-builder/slice-6-conversational-coordinator.md` |
| Slice 7 | Hatmax TUI | `feat/hatmax-tui` | `feat(slice-7): deliver the conversational Hatmax TUI` | `ops/default/report/slices/conversational-hatmax-builder/slice-7-hatmax-tui.md` |
| Slice 8 | Builder acceptance | `test/conversational-builder-acceptance` | `test(slice-8): validate the conversational Hatmax builder` | `ops/default/report/slices/conversational-hatmax-builder/slice-8-builder-acceptance.md` |

## Slice 1: Builder Contracts

### Purpose

Resolve bounded draft questions, reconcile affected approved specs, and
promote one non-conflicting contract before runtime work.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Resolve canonical composition ownership, bootstrap migration, Git initialization, root metadata, module-base configuration, target admission, and `main.go`; reconcile umbrella, Book, intent, execution, and CRUD contracts. | `docs(spec): reconcile application scaffold contracts` |
| T1.2 | Resolve TUI framework boundary, reset, retention, local state, dialogue limits, command compatibility, and recovery; reconcile the existing product-surface contract. | `docs(spec): reconcile conversational surface contracts` |
| T1.3 | Promote the scaffold, conversational surface, and state model together and validate terminology, links, acceptance, and precedence. | `docs(spec): approve the conversational Hatmax builder` |

### Validation

- `make docs-check`
- `git diff --check`

## Slice 2: Application Planning Kernel

### Purpose

Inventory pre-project targets, validate `create_application`, and produce a
sealed scaffold plan without writing files.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Add parent and target inventory, compatible-project detection, preserved-file and collision classification, fingerprints, remote evidence, and drift observations. | `feat(generator): inventory application targets` |
| T2.2 | Add application identity, target, optional enrichment, initial-feature units, deterministic normalization, module-path precedence, clarifications, and semantic validation. | `feat(generator): define application intents` |
| T2.3 | Add the application archetype and Book rules, expand scaffold obligations and composite units, serialize plans, and bind them to target state. | `feat(generator): plan application scaffolds` |

### Validation

- `go test ./generator/book/... ./generator/project/... ./generator/intent/... ./generator/plan/...`
- `go test -race ./generator/project/... ./generator/intent/... ./generator/plan/...`
- `go test ./generator/...`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 3: Scaffold Execution

### Purpose

Render, stage, publish, and evaluate one canonical compiling application from
an approved plan without product-surface integration.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Render the module, thin `main.go`, `internal/application`, configuration, lifecycle, Postgres, migration activation support, commands, ignore rules, and tests. | `feat(generator): render Hatmax application foundations` |
| T3.2 | Render the neutral router, middleware, templates, assets, landing page, and embedding surfaces without invented domain or documentation. | `feat(generator): render Hatmax web scaffolds` |
| T3.3 | Stage and publish targets atomically, preserve admitted files, enforce scaffold conformance, compile fixtures, classify test infrastructure, and cover rollback, collision, and idempotency. | `test(generator): enforce scaffold execution` |

### Validation

- `go test ./generator/execute/... ./generator/project/... ./generator/plan/...`
- `go test -race ./generator/execute/...`
- `go test ./generator/...`
- `make generator-scaffold-acceptance`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 4: Application Bootstrap Product

### Purpose

Expose application creation through existing interaction and Codex boundaries,
including one composite plan for requested initial features.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Extend strict interpretation for application identity, required clarifications, target evidence, niche context, and optional initial features without exposing files. | `feat(generator): interpret application creation` |
| T4.2 | Coordinate target inspection, one approval, scaffold execution, dependent feature units, conformance, compilation, best-effort tests, and reporting. | `feat(generator): coordinate application bootstrap` |
| T4.3 | Add deterministic and terminal acceptance for terse, detailed, clarified, conflicting, stale, cancelled, single-unit, and composite requests. | `test(generator): validate application bootstrap` |

### Validation

- `go test ./generator/eval/... ./generator/backend/codex/... ./generator/interaction/... ./internal/hatmaxcli/...`
- `go test -race ./generator/interaction/... ./generator/execute/...`
- `go test ./generator/...`
- `make generator-scaffold-acceptance`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 5: Conversation State

### Purpose

Define and persist bounded Hatmax-owned conversation state without granting
stored text or backend history project authority.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Add bounded dialogue results and turns while preserving strict mutation variants and rejecting tool or edit content. | `feat(generator): define conversational results` |
| T5.2 | Add conversation, proposed-operation, reset, stale, rebinding, retention, and store contracts with no durable approval. | `feat(generator): model persistent conversations` |
| T5.3 | Implement user-local versioned persistence with isolation, atomic updates, privacy bounds, thread references, pruning, and restart tests. | `feat(generator): persist local conversation state` |

### Validation

- `go test ./generator/eval/... ./generator/conversation/... ./internal/hatmaxstate/...`
- `go test -race ./generator/conversation/... ./internal/hatmaxstate/...`
- `go test ./generator/...`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 6: Conversational Coordinator

### Purpose

Coordinate open dialogue and closed Hatmax mutations as one resumable state
machine independent of terminal presentation.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T6.1 | Classify dialogue, candidate, clarification, unsupported, and plan-ready outcomes; collect required decisions and surface conversation-derived inputs before approval. | `feat(generator): coordinate conversational turns` |
| T6.2 | Resume project and pre-project conversations, reinspect state, rebind scaffolds, replace lost threads, invalidate plans, and support fresh conversations. | `feat(generator): resume Hatmax conversations` |
| T6.3 | Preserve diagnostics and revisable proposals across cancellation and failure while enforcing atomicity, retained-change reporting, invalid approval, and safe retry. | `feat(generator): recover conversational operations` |

### Validation

- `go test ./generator/conversation/... ./generator/interaction/... ./generator/backend/codex/...`
- `go test -race ./generator/conversation/... ./generator/interaction/...`
- `go test ./generator/...`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 7: Hatmax TUI

### Purpose

Deliver the `hm` conversational experience while preserving the same kernel
for headless automation.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T7.1 | Add the thin `hm` executable, TUI lifecycle, input, cancellation, resize, accessibility, and terminal restoration around the selected framework. | `feat(cli): add the hm command` |
| T7.2 | Present conversation, clarifications, plans, approval, progress, validation, unsupported requests, stale state, retained changes, and recovery without UI-owned semantics. | `feat(tui): present conversational Hatmax work` |
| T7.3 | Route headless subcommands through `hm`, retain tested `hatmax` compatibility, preserve exit behavior, and prove kernel parity. | `feat(cli): preserve Hatmax headless compatibility` |

### Validation

- `go test ./internal/hatmaxtui/... ./internal/hatmaxcli/... ./cmd/hm/... ./cmd/hatmax/...`
- `go test -race ./internal/hatmaxtui/... ./generator/interaction/... ./generator/conversation/...`
- `go test ./generator/... ./internal/hatmaxtui/... ./internal/hatmaxcli/...`
- `make generator-conversation-acceptance`
- `make vet`
- `make lint-strict`
- `git diff --check`

## Slice 8: Builder Acceptance

### Purpose

Prove the complete builder across deterministic, generated-project, terminal,
persistence, and live Codex boundaries, then document the usable product.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T8.1 | Exercise scaffold, initial features, reopen, evolution, plan revision, reset, drift, cancellation, recovery, blocked infrastructure, and off-domain rejection. | `test(generator): exercise conversational workflows` |
| T8.2 | Extend authenticated smoke for resident dialogue, application intent, clarification, approval, rebinding, thread reuse, and isolation without tools or source access. | `test(generator): validate live Hatmax conversations` |
| T8.3 | Update reference, User Guide, command migration, examples, and `Unreleased`; record validation boundaries and close the report. | `docs(generator): document conversational Hatmax` |

### Validation

- `make generator-scaffold-acceptance`
- `make generator-conversation-acceptance`
- `go test -race ./generator/... ./internal/hatmaxtui/... ./internal/hatmaxcli/...`
- `make generator-live-smoke`
- `make check`
- `make docs-check`
- `git diff --check`

## Delivery-Set Gate

After Slice 8 merges into `dev`, run the Slice 8 validation list on the exact
integrated commit. Authenticated smoke evidence may be recorded separately when
its user-authenticated prerequisite is unavailable to CI, but the delivery set
cannot claim that boundary passed without exact-commit evidence.

## Completion Criteria

- The new specs and all affected existing specs are approved and consistent.
- Every slice is delivered through its branch, report, pull request, and merge.
- A user can create a compiling canonical application from a parent directory.
- Optional initial features execute through one visible composite plan.
- `hm` provides resumable conversational Hatmax work without becoming a
  general coding harness.
- Conversation persistence never stores reusable approval or replaces source
  inspection.
- Headless and TUI operations use the same deterministic kernels.
- The exact integrated `dev` candidate passes the delivery-set gate.

This plan authorizes no `main` alignment, mirror update, tag, release, package
publication, deployment, or live database migration.

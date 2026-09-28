# Interactive Generator Product Surface Delivery Plan

Status: Approved
Delivery set: interactive-generator-product-surface
Slice strategy: layered
Reason: the product surface crosses a versioned interpreter contract, an
external JSON-RPC runtime, persistent session state, deterministic generator
coordination, and terminal interaction. Establishing and testing those layers
in dependency order keeps every slice reviewable without allowing the model
backend to absorb Hatmax-owned behavior.
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Specs:

- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/execution-and-conformance.md`
- `ops/default/spec/interactive-product-surface.md`

Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Base branch: `dev`
Planning base: `ec16a4df27c6bb3ee6004d6303195f4717da73de`
Active slice: Slice 1
Execution gate: satisfied by this approved planning commit on `dev`

## Objective

Deliver the first usable Hatmax generator command backed by a resident Codex
App Server. The command accepts a natural-language request, obtains one bounded
typed interpretation, prints the deterministic Hatmax plan, requires explicit
approval, executes only canonical Hatmax renderers, and reports conformance and
repository validation.

Codex remains an interpreter backend. It does not inspect the target project,
choose architecture, construct source, repair failures, approve plans, or
execute project changes.

## Delivered Prerequisites

- The `interactive-generator-plan-kernel` delivery set provides the Book,
  project inventory, typed intent, sealed plan, and `eval.Interpreter`
  boundary.
- The `interactive-generator-execution-conformance` delivery set provides
  deterministic renderers, atomic application, conformance, repository
  validation, and execution evidence.
- The interactive product-surface specification is approved on `dev`.
- The planning protocol baseline is Codex App Server v2 as exposed by
  `codex-cli 0.153.0`. Runtime compatibility is established through version
  and protocol behavior, not by silently accepting unknown responses.

## Planning Decisions

- The first backend is Codex App Server authenticated through the user's
  existing ChatGPT-managed Codex login. There is no API-key or provider
  fallback.
- Hatmax uses the backend default model and records the effective model. The
  product surface does not pin a model name.
- One interpretation turn has a two-minute default timeout. Cancellation sends
  `turn/interrupt`; timeout does not authorize a retry.
- One rejected schema result may be corrected through one additional turn in
  the same project thread. Clarification answers also continue that thread.
- A successful normal request uses one model turn regardless of the number of
  affected files or surfaces.
- Structural application failures roll back atomically. Conformance or
  repository-command failures after a successful source commit retain the
  bounded source changes and report them; the generator never creates a Git
  commit.
- The first terminal surface is line-oriented and interactive. It does not add
  a full-screen TUI framework or an automatic approval mode.
- Hatmax withholds the target-project path and contents from Codex, uses an
  isolated working directory, disables tool and external-context facilities,
  and rejects any observed tool activity. The shared user-level daemon is not
  represented as an operating-system confidentiality boundary.

## Scope

The delivery includes:

- a versioned, strict interpreter request and response contract;
- bounded clarification conversation and backend provenance;
- a Codex daemon/proxy lifecycle and App Server v2 JSON-RPC client;
- isolated project-thread discovery, start, resume, reset, and reuse;
- schema-constrained Codex interpretation with stable failure diagnostics;
- a backend-neutral interaction coordinator;
- visible plan serialization and digest-bound explicit approval;
- dispatch through the three existing canonical renderers;
- post-edit conformance, repository validation, and complete result reporting;
- the `hatmax generate` terminal command;
- deterministic fake-server acceptance coverage and an opt-in authenticated
  Codex smoke test;
- an `Unreleased` changelog entry for the completed product behavior.

It excludes:

- Pi or direct API adapters;
- API keys, provider fallback, or model selection UI;
- boxed Diataxis documentation generation;
- operations beyond `create_feature`, `add_field`, and `add_validation`;
- archetypes beyond `server_rendered_crud`;
- automatic plan approval, autonomous repair, or model-generated source;
- Git commits, branches, pushes, pull requests, publication, deployment, or
  live database migration;
- a network listener, remote App Server, or full-screen terminal framework.

## Architecture Boundaries

Package ownership is:

```text
generator/book/             Canonical Hatmax rules and selection
generator/project/          Target-project inspection and fingerprints
generator/intent/           Typed product intent and semantic validation
generator/plan/             Deterministic plan expansion and lifecycle
generator/eval/             Backend-neutral interpretation contracts and schema
generator/backend/codex/    Daemon, proxy, JSON-RPC, thread, and Codex adapter
generator/execute/          Canonical rendering, mutation, conformance, validation
generator/interaction/      End-to-end product lifecycle and stable outcomes
internal/hatmaxcli/         Terminal input, presentation, and exit mapping
cmd/hatmax/                 Thin executable entrypoint
```

`backend/codex` may implement `eval.Interpreter` but cannot import
`generator/execute` or select Book obligations. `interaction` may coordinate
the kernel and executor but cannot decode App Server protocol messages.
`internal/hatmaxcli` owns terminal wording only; stable statuses and
diagnostics remain in generator packages.

The Codex adapter owns only a minimal tested subset of App Server v2. It does
not vendor or expose the complete generated Codex protocol schema. Unknown
methods, server requests, tool items, oversized messages, and incompatible
daemon state fail closed.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Interpreter contracts | `feat/generator-interpreter-contracts` | `feat(slice-1): define interactive interpreter contracts` | `ops/default/report/slices/interactive-generator-product-surface/slice-1-interpreter-contracts.md` |
| Slice 2 | Codex App Server runtime | `feat/generator-codex-app-server` | `feat(slice-2): connect the Codex App Server runtime` | `ops/default/report/slices/interactive-generator-product-surface/slice-2-codex-app-server-runtime.md` |
| Slice 3 | Codex interpreter | `feat/generator-codex-interpreter` | `feat(slice-3): interpret Hatmax requests with Codex` | `ops/default/report/slices/interactive-generator-product-surface/slice-3-codex-interpreter.md` |
| Slice 4 | Interaction coordinator | `feat/generator-interaction-coordinator` | `feat(slice-4): coordinate interactive generation` | `ops/default/report/slices/interactive-generator-product-surface/slice-4-interaction-coordinator.md` |
| Slice 5 | Terminal product surface | `feat/generator-terminal-surface` | `feat(slice-5): expose the Hatmax generator command` | `ops/default/report/slices/interactive-generator-product-surface/slice-5-terminal-product-surface.md` |

## Slice 1: Interpreter Contracts

### Purpose

Turn the existing evaluation-only interpreter boundary into the strict,
versioned contract required by a persistent interactive backend without adding
Codex-specific behavior.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Extend `generator/eval` with bounded clarification exchanges, backend provenance, stable backend failures, request limits, and detached result ownership while preserving typed intent, clarification, and unsupported variants. | `feat(generator): define interactive interpreter contracts` |
| T1.2 | Add the strict versioned JSON output schema and decoding path for exactly one interpretation variant; reject unknown fields, contradictory variants, oversized results, plans, paths, commands, dependencies, and edits. | `feat(generator): constrain interpreter output` |
| T1.3 | Update the planning corpus and fixture interpreter for conversation and provenance, and cover malformed, ambiguous, injected, and equivalent requests without invoking a live model. | `test(generator): cover interactive interpretation contracts` |

### Validation

```sh
go test ./generator/eval/... ./generator/intent/... ./generator/plan/...
go test -race ./generator/eval/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 2: Codex App Server Runtime

### Purpose

Provide a bounded, reusable local connection to the installed Codex App Server
without yet assigning it authority over Hatmax interpretation or project
execution.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Add injected process and clock boundaries plus Codex executable discovery, CLI/server version checks, start-if-absent daemon behavior, and short-lived local proxy ownership. Never restart or stop an existing daemon. | `feat(generator): manage the Codex runtime` |
| T2.2 | Add a minimal App Server v2 JSON-RPC client for initialization, correlated requests, bounded notifications, cancellation, `turn/interrupt`, graceful proxy shutdown, and stable protocol errors using deterministic fake transports. | `feat(generator): speak the Codex App Server protocol` |
| T2.3 | Add opaque project-and-contract context identities, isolated working directories, thread list/start/resume behavior, backend-default model provenance, tool-free configuration, and fail-closed handling for server requests or tool activity. | `feat(generator): manage isolated Codex threads` |

### Validation

```sh
go test ./generator/backend/codex/...
go test -race ./generator/backend/codex/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 3: Codex Interpreter

### Purpose

Implement `eval.Interpreter` through one schema-constrained App Server turn and
prove that a resident daemon and isolated project thread can be reused safely.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Compile bounded `eval.Request` values into Codex input with the Hatmax interpreter instructions and versioned output schema; use the backend default model and two-minute default deadline. | `feat(generator): compile bounded Codex interpretation` |
| T3.2 | Implement the Codex interpreter turn lifecycle, final-agent-message extraction, strict result decoding, one schema-correction turn, clarification continuation, cancellation, tool-activity rejection, and stable provenance and error mapping. | `feat(generator): add the Codex interpreter` |
| T3.3 | Add fake App Server integration cases for daemon reuse, thread reuse, project isolation, authentication failure, incompatible protocol, timeout, interruption, malformed output, attempted tool use, and bounded event retention; add an opt-in authenticated live smoke target. | `test(generator): exercise Codex interpretation` |

### Validation

```sh
go test ./generator/backend/codex/... ./generator/eval/...
go test -race ./generator/backend/codex/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

The authenticated live smoke target is observational during this slice. It is
not part of ordinary unit tests and must not expose authentication material,
raw events, reasoning, or target-project data.

## Slice 4: Interaction Coordinator

### Purpose

Compose the delivered kernel and executor into one backend-neutral lifecycle
while keeping approval and every mutation decision under Hatmax control.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Add stable interaction states, outcomes, provenance, approval and clarification ports, limits, and diagnostics under `generator/interaction/`. | `feat(generator): define interactive generation lifecycle` |
| T4.2 | Coordinate project inspection, Book selection, bounded fingerprinting, interpretation and clarification, deterministic planning, canonical YAML presentation, digest-bound approval, and stale-plan rejection. | `feat(generator): coordinate intent and approval` |
| T4.3 | Dispatch approved plans to the existing canonical renderer, workspace, conformance, and repository-validation APIs; retain and report post-commit validation failures, prohibit Git operations, and cover complete fixture workflows. | `feat(generator): coordinate validated execution` |

### Validation

```sh
go test ./generator/interaction/... ./generator/execute/...
go test -race ./generator/interaction/... ./generator/execute/...
go test ./generator/...
make vet
make lint-strict
git diff --check
```

## Slice 5: Terminal Product Surface

### Purpose

Expose the approved lifecycle as the first usable `hatmax generate` command
and prove the complete product boundary.

### Tasks

| Task | Work | Expected commit |
| --- | --- | --- |
| T5.1 | Add a thin `cmd/hatmax` entrypoint and testable line-oriented terminal application supporting `hatmax generate "<request>"`, the current directory as project root, focused clarifications, plan display, explicit approval, cancellation, and stable exit statuses. | `feat(generator): add the Hatmax generate command` |
| T5.2 | Present bounded backend provenance, unsupported results, stale plans, execution changes, conformance, and exact repository-command evidence without credentials, reasoning, raw App Server events, or automatic approval. | `feat(generator): report interactive generation results` |
| T5.3 | Add end-to-end fake-backend acceptance cases for all three operations and principal failures, verify no Git or undeclared project effects, add the completed behavior to `CHANGELOG.md` under `Unreleased`, and run one authenticated Codex smoke test against the exact integrated candidate. | `test(generator): validate the terminal product surface` |

### Validation

```sh
go test ./cmd/hatmax/... ./internal/hatmaxcli/... ./generator/interaction/...
go test -race ./generator/interaction/... ./generator/backend/codex/...
go test ./generator/...
make check
make generator-live-smoke
git diff --check
```

`make check` must run through a process with Docker group membership because
the existing Hatmax integration tests use Testcontainers. The live smoke
requires an authenticated compatible Codex installation and reports the CLI,
App Server, effective model, thread reuse, and bounded result only.

## Delivery-Set Gate

After all five slices are merged into `dev`:

1. capture the exact `origin/dev` commit;
2. run `make check` with Docker access against that commit;
3. run the complete deterministic generator corpus with the race detector;
4. run `make generator-live-smoke` with the authenticated Codex subscription;
5. verify a second request reuses the daemon and same project thread;
6. verify a different fixture project does not reuse that thread;
7. verify rejected, cancelled, ambiguous, stale, and pre-commit failure paths
   leave the target project unchanged;
8. verify successful and post-commit validation-failure reports enumerate the
   exact retained changes;
9. close the tracker only when the exact candidate is green.

No `main` alignment, mirror update, tag, release, Pi adapter, API adapter, or
boxed documentation generation is implied by delivery-set closure.

## Completion Criteria

- `hatmax generate` accepts one natural-language request in a compatible
  Hatmax project.
- A compatible resident Codex App Server and isolated project thread are
  reused across commands.
- Normal interpretation uses one model turn and produces only a strict typed
  intent, focused clarification, or unsupported result.
- Codex receives bounded context without the target-project path or contents
  and any tool activity fails the interaction.
- Hatmax independently validates the intent and prints the deterministic plan
  before any edit.
- Only an explicit approval of the displayed plan digest permits execution.
- The three admitted operations dispatch through existing canonical Hatmax
  renderers and cannot add undeclared surfaces or dependencies.
- Final output distinguishes cancellation, unsupported intent, stale plans,
  backend failures, execution failures, and completed generation.
- Deterministic acceptance, the full repository gate, and one authenticated
  live Codex smoke pass for the exact integrated candidate.

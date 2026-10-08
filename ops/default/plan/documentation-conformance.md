<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Hatmax Documentation Conformance Delivery Plan

Date: 2026-10-08
Status: Approved
Approved: 2026-10-08
Delivery set: documentation-conformance
Concern: [Documentation conformance](../spec/documentation-conformance.md)
Tracker: [Delivery tracker](../tracker/documentation-conformance.md)
Base branch: `dev`
Planning base: `3ec58ea2207f71b8eddae39501e69485d2673470`
Active slice: Slice 1
Next slice: Slice 1
Execution gate: Open — independent slice-loop execution authorized on 2026-10-08
Slice strategy: layered
Reason: inventory and evidence tooling establish the audit boundary; independently
reviewable reader-workflow groups reconcile existing content; integrated acceptance
then verifies complete coverage on one immutable candidate.

## Authorization

The user requested planning on 2026-10-08 and then explicitly requested launching
this complete delivery set with the new slice-loop. The specification and plan
are approved, implementation is authorized, and Rebase + Fast-forward merges
into dev are delegated to this repository's independent controller.

The execution gate is open for Slice 1/T1.1/T1.2. Each slice has one
recorded branch, one canonical PR to dev and one report. The approved external
controller owns Rebase + Fast-forward integration and immediate continuation
through this set; no separate reviewer agent is introduced.

## Baseline and Prerequisites

The planning base contains the recent authentication documentation correction,
not a complete documentation audit. Existing `make docs-check` checks structure,
local links, example compilation and whitespace; it does not establish that all
published commands and workflows have actually run.

The repository uses README documentation entrypoints and has no nightly workflow.
Keep that local convention. Use Go 1.27.1. Root packages currently discovered are
app, auth, config, crypto, db, fake, format, htmx, i18n, image, log, mailer,
middleware, modal, model, pagination, pubsub, render, scheduler, seed, settings,
slug, telemetry, testhelper, ui, validation and web. Inventory commands, nested
packages, examples and generator areas separately; the list is a seed, not an
exhaustive completion assertion.

At activation, verify clean dev, canonical origin/dev, current PR/worktree state
and the absence of a competing controller in this repository. Record the actual
activation commit and source/tool identities. Source drift must be reconciled
against the inventory before a coverage item or old receipt can count as current.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Coverage and validation | `docs/hatmax-documentation-coverage` | `docs(slice-1): inventory Hatmax documentation coverage` | `ops/default/report/slices/documentation-conformance/slice-1-coverage-and-validation.md` |
| Slice 2 | Runtime and presentation | `docs/hatmax-runtime-presentation` | `docs(slice-2): reconcile runtime and presentation guidance` | `ops/default/report/slices/documentation-conformance/slice-2-runtime-and-presentation.md` |
| Slice 3 | Data and configuration | `docs/hatmax-data-configuration` | `docs(slice-3): reconcile data and configuration guidance` | `ops/default/report/slices/documentation-conformance/slice-3-data-and-configuration.md` |
| Slice 4 | Identity and recovery | `docs/hatmax-identity-recovery` | `docs(slice-4): verify identity and recovery documentation` | `ops/default/report/slices/documentation-conformance/slice-4-identity-and-recovery.md` |
| Slice 5 | Infrastructure and helpers | `docs/hatmax-infrastructure-helpers` | `docs(slice-5): reconcile infrastructure and helper guidance` | `ops/default/report/slices/documentation-conformance/slice-5-infrastructure-and-helpers.md` |
| Slice 6 | Generator and assisted workflows | `docs/hatmax-generator-guidance` | `docs(slice-6): reconcile generator and assisted workflow guidance` | `ops/default/report/slices/documentation-conformance/slice-6-generator-and-assisted-workflows.md` |
| Slice 7 | Integrated documentation acceptance | `docs/hatmax-documentation-acceptance` | `docs(slice-7): establish complete documentation acceptance` | `ops/default/report/slices/documentation-conformance/slice-7-integrated-documentation-acceptance.md` |

## Task Map

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Inventory every delivered package, executable, example, Book/help surface and documentation page; establish source-bound coverage and four-quadrant ownership. | `docs: inventory Hatmax documentation surfaces` |
| T1.2 | Create the repository-owned conformance entrypoint, structural/source checks and negative controls; support Slice 1 without requiring future slice evidence. | `test(docs): establish Hatmax conformance checks` |
| T2.1 | Reconcile application assembly, lifecycle, HTTP/web, middleware, render, HTMX, UI, modal, format, pagination and i18n across the learning path and supporting quadrants. | `docs: reconcile runtime and presentation guidance` |
| T2.2 | Compile and execute the documented health/request/page/form examples in isolated composition; verify redirects, rendering and shutdown outcomes. | `test(docs): verify runtime and presentation examples` |
| T3.1 | Reconcile config, settings, db, model, validation, seed and log contracts, precedence, migrations, persistence and resource ownership. | `docs: reconcile data and configuration guidance` |
| T3.2 | Execute PostgreSQL/migration and runtime-setting procedures against fresh owned storage; check published configuration through real validators and compiled examples. | `test(docs): verify data and configuration examples` |
| T4.1 | Audit the complete auth and crypto surface, including password, admission, proof, sessions, enrollment, factor changes, mailbox/reset and key handling; retain accurate recent additions. | `docs: reconcile identity and recovery guidance` |
| T4.2 | Execute the documented Ticked authentication/recovery compositions with real local storage, captured mail and browser proof; validate crypto snippets in their stated context. | `test(docs): verify identity and recovery examples` |
| T5.1 | Reconcile mailer, image, pubsub, scheduler, telemetry, testhelper, fake and slug documentation and related application-services/testing chapters. | `docs: reconcile infrastructure and helper guidance` |
| T5.2 | Execute documented local mail/image/event/job/helper examples with owned sinks and resources; check error, cancellation, cleanup and scheduling claims where documented. | `test(docs): verify infrastructure and helper examples` |
| T6.1 | Reconcile CLI/TUI/help, conversation, intent/plan/execute, project inspection, Book capability claims and generated documentation with delivered behavior. | `docs: reconcile generator and assisted workflow guidance` |
| T6.2 | Run supported documented generator commands and interactions against fresh isolated projects; build generated output and verify actual conformance/navigation and closed unsupported cases. | `test(docs): verify generator documentation workflows` |
| T7.1 | Complete all inventory rows, reconcile cross-quadrant terminology and task/source evidence, and perform the complete published learning/task walkthrough sweep. | `docs: reconcile complete Hatmax documentation coverage` |
| T7.2 | Finalize the self-contained immutable documentation gate and DC01–DC10 acceptance mapping; leave integrated gate results pending until the controller executes them. | `test(docs): bind Hatmax documentation acceptance` |

## Coverage and Evidence Binding

Slice 1 creates `ops/default/report/documentation-conformance-coverage.md`.
Each row identifies a stable surface/page ID, source owner and inspected revision,
reader need/quadrant, owning slice, relevant page/anchor, verification method,
receipt and status. Discover public surfaces and tracked pages independently,
then reconcile the two lists. A missing page or undocumented surface is a finding,
not grounds for excluding it from the inventory.

Every runnable code block or procedure has a stable example ID. Evidence records
the exact command, relevant source/snippet digest, tool and configuration identity,
expected observable result, actual exit/outcome and safe diagnostic references.
Classify fragments explicitly and validate them within a real compiled or checked
composition. Source-reviewed contextual deployment guidance may retain a precise
local-versus-external assurance boundary; it must not claim a live deployment check.
A runnable local example without execution evidence remains unverified.

Checked status belongs only to the exact inspected content and inputs. Unchanged
receipts may be reused when those identities match; changed source, examples or
configuration invalidate affected receipts. A cache entry or an empty test match
cannot supply evidence. Every report distinguishes compilation, local real
execution, fixture adapters and source-reviewed external-provider constraints.

## Reproducible Slice Gate

T1.2 creates `scripts/check-documentation-conformance.sh` and supporting checks.
Accept `slice N` only for implemented modes. Slice gates validate completed groups
through N; they do not require future groups to be marked verified. The default
integrated mode stays unavailable until the final slice implements it.

Each slice entrypoint runs `make docs-check`, `make source-license-check` and
`make lint-strict`, plus its active coverage and example checks. Do not mutate
badges or invoke `make ci`. The existing docs gate compiles example packages;
new execution receipts must establish the additional documented outcomes.

The runner resolves required tools and explicit overrides without storing machine
paths. Any execution needing a database, SMTP, HTTP server, generated project or
browser uses fresh resources in that invocation's ignored fixture directory.
It provisions its own keys, roles, schemas, socket/ports and cookie contexts.
It starts no production service and sends mail only to its owned local sink.
On exit it stops only its owned fixtures and retains safe diagnostics. Parallel
projects must never share mutable fixture names, ports or process ownership.

Use deterministic subprocess and browser bounds with explicit cleanup. Do not
impose a controller timer that terminates an active validation. Missing tools,
unknown completion or unexpected source drift block rather than supply a pass.

For every PR, the required validation prefix is the exact slice command below.
Additional successful focused commands follow it; `git diff --check` is last.
Record success against the exact clean pushed PR head after its reference commit.

### Slice 1: Coverage and validation

Acceptance: DC01, DC02.

Reconcile the inventory with tracked Go packages, commands, Book content and all product Markdown. Demonstrate rejection of a missing surface, broken anchor and unaccounted page.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 1`

### Slice 2: Runtime and presentation

Acceptance: DC03, DC08.

Validate current constructors/options, route boundaries and observed page/form outcomes. Contextual fragments must name their owning complete example.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 2`

### Slice 3: Data and configuration

Acceptance: DC04, DC08.

Check effective values, schema ownership, query/transaction contracts, migration ordering, setting updates and safe diagnostics. Preserve actual historical migration behavior.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 3`

### Slice 4: Identity and recovery

Acceptance: DC05, DC08.

Establish actual proof and expected denials without seeded authority. Verify freshness, cookie changes, factor preservation and lost-response guidance; do not equate package test success with every published instruction.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 4`

### Slice 5: Infrastructure and helpers

Acceptance: DC06, DC08.

Use captured mail and local files rather than live external accounts. Verify public names, defaults, limits and observable results; classify provider-only guidance with a precise source-review boundary.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 5`

### Slice 6: Generator and assisted workflows

Acceptance: DC07, DC08.

Identify deterministic versus provider-backed evidence. Exercise documented public options and generated artifacts. A fixture backend does not prove live provider behavior; document prerequisites and bound the evidence accordingly.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 6`

### Slice 7: Integrated documentation acceptance

Acceptance: DC01–DC10.

Require complete source/page coverage, no unresolved or unverified claims, actual runnable-example receipts and one immutable integrated dev gate before closure.

Required PR validation prefix:

`GOWORK=off scripts/check-documentation-conformance.sh slice 7`

## Independent Controller Binding

Controller set/configuration name: `hatmax-documentation-conformance`.
Suggested view: `slice-watch hatmax-documentation-conformance`.

Before the first run, initialize a fresh local configuration
outside the Git repository with this plan/tracker and full gate:

`GOWORK=off scripts/check-documentation-conformance.sh`

Freeze the exact slice map and per-slice required prefixes from this plan.
Freeze closure paths to this specification, plan, tracker, the coverage/acceptance
records and the 7 canonical reports. The controller configuration,
checkpoint, worker conversation, merge intent and monitor belong only to this
repository/set. Never reuse a previous completed run, another project's session,
a shared tracker or mutable cross-project acceptance record.

Verify the current loop executable and its actual supported options at launch.
Report its exact configuration/run ID and literal view command. Independent
repository loops may run concurrently with distinct fixtures and views; one
repository admits only its own active controller. Recovery selects an exact
run/configuration and observes retained PR/checkpoint state before continuation.

## Canonical PR Contract

Use exactly `## Summary`, `## Changes`, `## Validation`, in that order.
Summary is one concise outcome paragraph. Changes lists concrete delivered
changes and ends with the immutable link to that slice report's introduction
commit. Validation lists only exact successful commands, with whitespace last.
No roadmap narration or obvious slice-boundary disclaimers belong in the body.
Verify exact title/base/branch/head/report and canonical merge result.

## Final Integration and Closure

After all 7 slice PRs and report references are integrated, capture one clean
immutable dev candidate identical to canonical origin/dev. The controller runs
the repository-owned default documentation gate once against that candidate.
It checks the full inventory, all pages/anchors, effective examples and all
accepted walkthrough groups, plus source/dependency/tool identity and required
repository documentation checks. This is the full gate for this documentation
set; do not label it the complete product runtime test suite.

Create `ops/default/report/documentation-conformance-acceptance.md` with every
criterion mapped to actual evidence, the exact tested candidate/command and
precise assurance limits. No unresolved, failed or unverified runnable example
may be converted to passed for closure. A real runtime defect uses its own
repository ticket/spec and blocks the affected claim instead of expanding this set.

A repository correction to the integrated gate uses the recorded contingency
branch `fix/documentation-conformance-validation` and one canonical PR to dev,
with focused correction checks before the controller revalidates. An environmental
failure preserves state and requires no empty correction PR.

Only after the gate passes may the worker complete the spec/plan/tracker,
deliver all reports and record acceptance. Any later closure commit may change
only frozen operational closure paths and must leave the tested product docs,
examples and validation code unchanged. Do not rerun expensive accepted journeys
for those metadata-only changes. Completion applies to the validated source
revision, not unbounded future code changes.

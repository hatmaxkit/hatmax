<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Infrastructure and Helpers

Status: delivered
Integrated candidate: `090e260378cbb47734f6202be360923a9db1c3ac`
Integrated acceptance: [Passed controller evidence](../../documentation-conformance-acceptance.md)
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-infrastructure-helpers`
PR: `#116`
Activation revision: `414b9e071f95b07f531aae5808a8eb903c066a16`
Documentation task: `a94d6a41afd83e55c7cd02fe70b36238846c9e04`
Validation task: `38eb310994d7abf1731bd3aeb2ccb14d8a628ae4`
Report introduction: `b1b77bb68e9e0c3c25b87f5b2ed2a64076dc8122`
Canonical PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/116
Merged revision: `b29aa327d3b8a815b79c894571438ce2a6f44144`

## Purpose

Reconcile mail, image, event, scheduling, telemetry and testing guidance with
current contracts and establish actual local workflow evidence.

## Delivered Behavior

T5.1 corrects constructor contexts, repeated declarations and incomplete
quickstarts. Eight package examples now provide complete Go programs, functions
or tests. Supporting how-to, reference and learning-path guidance distinguishes
construction-time mail settings from live reload, validation from delivery,
JSON-decoded events from original Go values, and persisted scheduler retries
from recurring slots. Image guidance records reader ownership, output format,
local copy/cancellation limits and partial-file cleanup. Telemetry describes
byte truncation, absent redaction and unbounded distinct grouping. Test helpers
document environment-based selection, consumer-owned pools and actual cleanup
limitations. Fake and slug notes state attempted-call, pointer and collision
contracts.

T5.2 compiles all 35 exact Go fragments and executes eight complete package
examples plus published local workflows. Owned SMTP captures matching message
content; local PNG readback verifies dimensions, extension, URL and deletion.
Real PostgreSQL exercises independent subscriber acknowledgement, pending
delivery after restart, recurring job slots, persistent retry budgets, panic and
unknown-handler outcomes, cancellation and joined shutdown. Telemetry records
and drains actual HTTP/panic observations. Fake capture/reset, slug outputs,
helper schema selection and externally inspected removal also pass.

## Implementation Notes

The dedicated branch starts at current canonical dev after PR #115 closure.
The 451-row inventory preserves slice identities. Slice 5 supplies 92 classified
bindings: 14 package suites, 11 compilation-only contexts, 38 executed bindings
and 29 source inspections. Exact source blocks supply generated contexts and
workflows; typed application adapters make their owning context explicit. Both
published helper commands run in an isolated module containing the exact test.

Each invocation owns its PostgreSQL cluster, schemas, SMTP listener and files.
Go owns fixture generation, message/image assertions and process coordination;
shell owns bounded cluster setup and teardown. Receipts bind the clean head,
source/snippet digests, compiler, native tool binaries, generated fixture inputs,
ordered commands and diagnostic digests. Missing, stale, duplicate, failed,
cached or compile-only substitutes cannot establish execution. Cumulative
Slice 2–4 checks precede the new Slice 5 run.

## Contracts Added or Changed

Documentation reflects existing APIs and resource ownership; toolkit behavior
and module dependencies are unchanged. Scheduler retry configuration is applied;
only its pause setting is consulted dynamically. Telemetry modes describe
application exporter policy and do not activate collection or transmission.
The slice runner now accepts Slice 5 and reconciles source-bound infrastructure
receipts after owned fixture shutdown.

## Files of Interest

- [Mailer notes](../../../../../mailer/readme.md)
- [Image workflow](../../../../../docs/how-to/store-images/README.md)
- [Pubsub notes](../../../../../pubsub/readme.md)
- [Scheduler workflow](../../../../../docs/how-to/run-background-jobs/README.md)
- [Telemetry reference](../../../../../docs/reference/telemetry/README.md)
- [Test helper workflow](../../../../../docs/how-to/test-with-postgres/README.md)
- [Native workflow fixture](../../../../../scripts/documentation-conformance/infrastructure_fixture.go)
- [Evidence reconciliation](../../../../../scripts/documentation-conformance/infrastructure_evidence.go)
- [Coverage record](../../documentation-conformance-coverage.md)

## Validation

Focused T5.1 checks passed before its task commit:

- `GOWORK=off make docs-check` — links, structure and existing example compilation.
- `GOWORK=off make source-license-check` — 924 headers and 113 content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 current identities and navigation.
- `GOWORK=off GOFLAGS=-p=2 go test -run '^$' ./...` — eight exact first Go blocks compiled in the ignored `.tmp/documentation-conformance/source-audit/contexts` module with a local source replacement; no test execution assertion.
- `git diff --check` — passed.

Focused T5.2 checks passed:

- `GOWORK=off make docs-check` — structure, links, compilation and whitespace.
- `GOWORK=off make source-license-check` — 930 headers and 113 content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 identities.
- `GOWORK=off go test -count=1 -timeout=2m ./scripts/documentation-conformance` — evidence and context negative controls.
- `bash -n scripts/check-documentation-conformance.sh scripts/documentation-conformance/data-fixture.sh` — passed.
- `PATH="$PWD/.tmp/documentation-conformance/tools:$PATH" GOWORK=off GOFLAGS=-p=2 scripts/documentation-conformance/data-fixture.sh "$fixture" go run ./scripts/documentation-conformance infrastructure "$fixture"` — 92 bindings in a fresh owned development fixture.

Fixture failures exposed two harness assumptions:
configuration loading requires a non-empty argument vector, and broker payload
storage needs UTF-8 decoding before JSON inspection. Corrected fixtures passed;
no production correction was required.

The controller passed the final ordered checks against clean pushed head
`b29aa327d3b8a815b79c894571438ce2a6f44144`, including canonical PR references:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 5`
- `git diff --check`

The gate reconciled 451 identities and 141 runtime, 89 data, 50 identity and
92 infrastructure bindings. Owned sinks, processes, schemas and clusters stopped.
Canonical Forgejo and Acta confirm PR #116 merged at the tested revision.

## Risks and Follow-ups

DC06/DC08 integrated acceptance and DC09's constrained native-toolchain execution
passed in the controller's full documentation gate on the candidate above.
The earlier exact-head receipt retains its Slice 5 validation meaning. SMTP capture cannot
establish inbox acceptance; S3 configuration and package fixtures cannot establish
cloud account access. This gate executes the helper server path; its container
path remains source-inspected. Typed application fragments establish compilation
only. Source, head, configuration or tool drift requires renewed execution.

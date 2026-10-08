<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Infrastructure and Helpers

Status: drafting
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-infrastructure-helpers`
PR: pending
Activation revision: `414b9e071f95b07f531aae5808a8eb903c066a16`
Documentation task: `a94d6a41afd83e55c7cd02fe70b36238846c9e04`

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

T5.2 execution evidence and the Slice 5 gate remain pending. No workflow is
marked executed on the strength of this documentation task's compilation check.

## Implementation Notes

The dedicated branch starts at current canonical dev after PR #115 closure.
The 451-row coverage inventory preserves prior slice identities and updates only
changed Slice 5 source/snippet digests, with their behavioral receipts pending.
Native-toolchain closure wording and exact slice identities remain unchanged.

The next task must compile all contextual Go fragments and execute published
local mail/image/event/job/helper workflows using owned resources. Required
outcomes include captured mail, image readback and deletion, named subscriber
retry/restart, durable scheduled-slot transitions and retry budgets, cancellation,
telemetry drain and test-schema removal. Provider-only configuration retains its
source-review boundary; local adapters cannot establish external inbox or cloud
account acceptance.

## Contracts Added or Changed

Documentation reflects existing APIs and resource ownership; toolkit behavior
and module dependencies are unchanged. Scheduler retry configuration is applied;
only its pause setting is consulted dynamically. Telemetry modes describe
application exporter policy and do not activate collection or transmission.

## Files of Interest

- [Mailer notes](../../../../../mailer/readme.md)
- [Image workflow](../../../../../docs/how-to/store-images/README.md)
- [Pubsub notes](../../../../../pubsub/readme.md)
- [Scheduler workflow](../../../../../docs/how-to/run-background-jobs/README.md)
- [Telemetry reference](../../../../../docs/reference/telemetry/README.md)
- [Test helper workflow](../../../../../docs/how-to/test-with-postgres/README.md)
- [Coverage record](../../documentation-conformance-coverage.md)

## Validation

Focused T5.1 checks passed before its task commit:

- `GOWORK=off make docs-check` — links, structure and existing example compilation.
- `GOWORK=off make source-license-check` — 924 headers and 113 content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 current identities and navigation.
- `GOWORK=off GOFLAGS=-p=2 go test -run '^$' ./...` — eight exact first Go blocks compiled in the ignored `.tmp/documentation-conformance/source-audit/contexts` module with a local source replacement; no test execution assertion.
- `git diff --check` — passed.

The final ordered controller checks are pending until T5.2, canonical PR
references and the final clean pushed head are complete:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 5`
- `git diff --check`

## Risks and Follow-ups

T5.2 and DC06/DC08 acceptance are pending. DC09's required constrained native
toolchain execution remains part of integrated closure under the generic
approved policy. Local capture,
filesystem, database and fixture endpoint evidence must retain their actual
assurance limits. Source, head, configuration or tool drift requires renewed
execution evidence.

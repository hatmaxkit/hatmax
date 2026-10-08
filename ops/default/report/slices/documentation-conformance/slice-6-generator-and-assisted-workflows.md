<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 6: Generator and Assisted Workflows

Status: delivered
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-generator-guidance`
PR: `#117`
Activation revision: `c7e9858d8726e0a5f1ed56f41c2daee4e59cebff`
Documentation task: `0a70f3b71cd69db25dfaae0a62224a0f6eef087d`
Validation task: `cdd3e8888c4a34bd253dc5503fe4f9bd37c599ba`
Report introduction: `0a70f3b71cd69db25dfaae0a62224a0f6eef087d`

## Purpose

Reconcile generator, terminal, conversation, Book and generated documentation
guidance with delivered behavior and verify local workflows with explicit limits.

## Delivered Behavior

The guide installs current `hm` from source; published v0.5.0 has no `cmd/hm`.
Reference and guide describe actual usage, playground tools, Book compatibility,
managed documentation, retention/privacy, backend isolation and validation
outcomes. Real workflow failures now appear in product guidance and open tickets.

Native fixtures exercise both command entrypoints, exact installation and shell
procedures, owned PTY help/quit, conversation selection and no-run playground
preparation. Production coordinators with fixture interpretation create real
published-dependency scaffolds, rebind/resume state and evolve canonical existing
projects. Feature creation, four-quadrant documentation, user-text preservation,
approval/cancellation, reset, stale plans and unsupported capabilities are checked.

The full create-then-evolve walkthrough and two timestamp examples remain blocked.
Fresh scaffolds cannot resolve their follow-up composition root. Required
canonical timestamps fail SQLC row-mapping compilation after retaining changes.
Minimum-length validation compiles and passes Go tests, then fails generated
model-test lint. No runtime correction or successful-journey claim is delivered.

## Implementation Notes

The dedicated branch follows verified PR #116 closure. The 451-row inventory
preserves exact slice identities. Slice 6 reconciles 63 bindings: 12 uncached,
non-empty package suites, 14 executed bindings, 34 source inspections and three
blocked rows. Passing negative-outcome assertions preserve actual failures;
they do not mark the affected operations completed.

Go supplies fixture interpretation and assertions; shell owns bounded PostgreSQL
setup/stop. Exact published shell blocks supply native command procedures.
Generated existing-project contexts explicitly replace the toolkit with this
candidate; the separate bare scaffold retains the published module unchanged.
Receipts bind head, source/snippets, compiler, native tools, canonical fixture
inputs, built commands and diagnostic digests. Incomplete, cached, altered or
unbound workflow evidence is rejected. Cumulative Slice 2–5 checks precede Slice 6.

## Contracts Added or Changed

Toolkit runtime and dependencies are unchanged. Documentation records generated
ownership markers, local-file conformance limits and state without general
secret redaction. The evidence runner accepts Slice 6 and retains `blocked`
rows as unresolved. Backend protocol and interpreter fixtures establish no
production model or authenticated provider acceptance. Native-toolchain closure
retains the generic approved requirement.

## Files of Interest

- [Generator reference](../../../../../docs/reference/generator/README.md)
- [Assisted generation](../../../../../docs/tutorials/user-guide/assisted-generation.md)
- [Native workflow fixture](../../../../../scripts/documentation-conformance/generator_fixture.go)
- [Evidence reconciliation](../../../../../scripts/documentation-conformance/generator_evidence.go)
- [Coverage record](../../documentation-conformance-coverage.md)
- [Scaffold composition defect](../../../ticket/solved/20261008134713-generator-scaffold-evolution.md)
- [Timestamp row-mapping defect](../../../ticket/solved/20261008135600-generator-timestamp-row-mapping.md)
- [Validation lint defect](../../../ticket/solved/20261008135846-generator-validation-test-lint.md)

## Validation

Focused T6.1 checks passed:

- `GOWORK=off make docs-check` — structure, links, compilation and whitespace.
- `GOWORK=off make source-license-check` — headers and content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 identities.
- `GOBIN="$PWD/.tmp/documentation-conformance/install" GOWORK=off go install ./cmd/hm` — source installation into an owned destination.

Focused T6.2 control tests and the generator runner passed in fresh owned
PostgreSQL fixtures. All three native workflow tests passed their assertions,
including required reproduction of the three blocked outcomes. SQLC 1.30.0 and
1.31.1 both exposed the timestamp compiler defect. Owned clusters stopped.
These results establish evidence classification, not a passing complete journey.

Focused repository checks passed before the validation task commit:

- `GOWORK=off make docs-check`
- `GOWORK=off make source-license-check`
- `GOWORK=off make lint-strict`
- `GOWORK=off go run ./scripts/documentation-conformance check`
- `GOWORK=off go test -count=1 -timeout=2m ./scripts/documentation-conformance`
- `bash -n scripts/check-documentation-conformance.sh scripts/documentation-conformance/data-fixture.sh`
- `git diff --check`

The controller passed final validation on clean pushed head
`b1c4a8c37278d400bea93fa2b96307b7cb8bf54d`. Its ordered evidence is authoritative:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 6`
- `git diff --check`

Canonical PR #117 merged that head into dev through Rebase + Fast-forward.
Forgejo and Acta confirm the merge. All cumulative evidence checks passed and
owned fixtures stopped; the three blocked rows remain unresolved.

## Risks and Follow-ups

Three open runtime tickets block affected DC07/DC08 claims. Their fixes require
separate authorization; this documentation set must not count the three blocked
rows as verified success for closure. DC09 constrained native-toolchain execution
and integrated acceptance remain pending. Fixture interpretation cannot prove
live model acceptance. Source, head, tool or configuration drift invalidates
execution evidence.

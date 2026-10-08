<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Documentation Conformance Acceptance

Status: delivered
Completed: 2026-10-08
Delivery set: documentation-conformance
Specification: [Documentation conformance](../spec/documentation-conformance.md)
Plan: [Delivery plan](../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../tracker/documentation-conformance.md)
Coverage: [Source-bound inventory](documentation-conformance-coverage.md)
Integrated candidate: `090e260378cbb47734f6202be360923a9db1c3ac`
Integrated command: `GOWORK=off scripts/check-documentation-conformance.sh`
Integrated result: passed; zero blocked rows
Controller configuration: `hatmax-documentation-conformance`
Controller evidence: `checks/090e260378cbb47734f6202be360923a9db1c3ac.json` and corresponding `.log` in the independent controller state
Evidence JSON SHA-256: `6b7d45e31803b0cb1353f0ab6dfc918c25b04fed55b244b12ffdeba4ef509d70`
Evidence log SHA-256: `f967535a6068bea6a674ebee7cbb7ffd860cdb647681a636c01604122f5fca4e`

## Acceptance Mapping

Each row names the durable check that supplies evidence on the integrated
candidate. Earlier focused receipts establish slice evidence only. The controller
passed the default gate against the exact clean canonical dev revision above;
its evidence establishes the integrated outcomes below. Closure updates only
the configured operational documents and preserves the tested product inputs.

| Criterion | Bound evidence | Current acceptance |
| --- | --- | --- |
| DC01 | Independent source/page/block discovery, complete digest reconciliation and coverage proof cardinality | passed; 451 identities reconciled |
| DC02 | Four-quadrant graph reachability, local links, anchors and T7.1 index/User Guide intent comparison | passed |
| DC03 | Slice 2 uncached package tests, exact contextual compilation, template rendering, source-bound bootstrap/Guide HTTP and shutdown receipts | passed; 141 bindings |
| DC04 | Slice 3 actual config validators, SQLC, fresh PostgreSQL schema/migrations/settings/seeding and Guide persistence receipts | passed; 89 bindings |
| DC05 | Slice 4 exact crypto contexts, real production browser/race journeys and owned Ticked setup/identity/restart/shutdown receipts | passed; 50 bindings |
| DC06 | Slice 5 exact fragments, native owned SMTP/filesystem/database workflows and infrastructure/helper package receipts | passed; 92 bindings |
| DC07 | Slice 6 production CLI/TUI/local state, Book source comparison, bare scaffold, canonical feature and managed documentation receipts | passed; 63 completed bindings |
| DC08 | Every owning group's exact digest/method/outcome proofs; Slice 7 root examples bound to executed bootstrap, native PTY and production CLI fixture | passed; all workflow proofs and 16 entrypoint bindings |
| DC09 | Default gate on clean canonical dev; cumulative execution with allowlist-only tool lookup, executable identities and local Go 1.27.1; complete acceptance rejects unresolved coverage | passed; 72 native executable identities, zero blocked rows |
| DC10 | Exact candidate/command/result here, canonical slice reports and immutable task/PR references, retained independent controller checkpoint | passed; approved operational closure preserves the tested inputs |

## Integrated Validation

The controller's exact successful command was:

- `GOWORK=off scripts/check-documentation-conformance.sh`

It ran `make docs-check`, `make source-license-check` and `make lint-strict`
in that order, then source/navigation reconciliation, uncached checker tests and
all five local workflow groups. Each group produced fresh exact-head receipts
and passed independent evidence reconciliation. Complete acceptance passed
with zero blocked rows. The integrated receipt binds the five group receipts,
16 entrypoint proofs and native manifest. Owned database, server, browser and
mail fixtures stopped after their checks; existing services were not stopped.

## Native Toolchain Evidence Contract

Current validators and workflow helpers use Go and shell where sufficient.
The gate resolves native Go 1.27.1 and formatter images, pins local toolchain
selection, and replaces inherited executable search directories with a finite
allowlist. It retains required shell/Make utilities, compiler tools, PostgreSQL,
SQLC, lint, Chromium and the product's native Codex executable. Go's runner can
prepend its own verified Go/formatter directory. Browser JavaScript retains its
product role. No general auxiliary-runtime directory is added to subprocess lookup.

The gate executes repository checks and every local workflow group in that
context. It records tool image digests, exact source/head/compiler identities,
command results, safe diagnostic digests, observations and owned shutdown.
The integrated receipt binds all five preceding execution receipts and the
native manifest. A missing tool, unexpected lookup directory, changed image,
stale fixture, missing named test or unresolved row rejects acceptance.
Constrained lookup is the tested dependency boundary; it is not a filesystem
sandbox. The controller's final execution passed with 72 executable identities
and no inherited lookup directories, satisfying the approved generic native
tooling requirement through actual workflow execution.

## Resolved Claims and Assurance Limits

The assisted-generation walkthrough and timestamp request blocks 6 and 9
previously failed. Their separately approved runtime corrections are recorded in
[TKT-20261008134713](../ticket/solved/20261008134713-generator-scaffold-evolution.md),
[TKT-20261008135600](../ticket/solved/20261008135600-generator-timestamp-row-mapping.md)
and [TKT-20261008135846](../ticket/solved/20261008135846-generator-validation-test-lint.md).
The corrections preserve delegated application composition, SQLC table row
types and lint-compliant generated validation tests. Native regressions now
require successful bare/initial-feature evolution, timestamp persistence and
validation checks. Documentation workflow controls require completed operations;
the earlier failing outcomes cannot establish current acceptance. The integrated
gate passed the corrected workflows; no unresolved coverage row remains.

Generator interpretation uses bounded Go fixtures around production coordinators
and CLI parsing. It proves local admission, approval, mutation, persistence and
conformance; it establishes no authenticated production model acceptance. External
provider guidance retains explicit prerequisites and source-inspection limits.
Contextual snippets and package tests retain their classified meaning instead
of becoming complete application or remote-provider workflows. Historical Book
illustrations are source-inspected positive/negative examples, not standalone
programs. Detailed boundaries remain in the owning slice reports and receipts.

This is the documentation-set gate for one immutable revision. It is not the
complete product runtime test suite and cannot establish future conformance
after source, dependency, tool or documentation changes.

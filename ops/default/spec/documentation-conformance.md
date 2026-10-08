<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Hatmax Documentation Conformance

Date: 2026-10-08
Status: Approved
Kind: Documentation assurance concern
Planning scope: Requested on 2026-10-08
Approved: 2026-10-08
Execution authorization: Complete this delivery set through its independent slice-loop, including Rebase + Fast-forward merges into dev
Implementation: Authorized
Plan: [Delivery plan](../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../tracker/documentation-conformance.md)
Related contract: [Boxed documentation](boxed-diataxis-documentation.md)
Related contract: [Hatmax Book](hatmax-book.md)

## Purpose

Make the complete delivered Hatmax documentation accurate and usable against
one identified repository revision. A successful link check or a review of
authentication alone cannot establish that the complete toolkit documentation
is current. Completion requires surface coverage, source comparison and actual
example evidence across tutorials, how-to guides, reference and explanation.

## Reader Outcomes

The User Guide gives developers a connected learning path through application
assembly and assisted generation. Task guides produce specific verifiable
results. Reference states current public contracts and settings. Explanation
describes the implemented boundaries and tradeoffs without promising capabilities
that exist only in a specification.

## Coverage

Inventory every public package, executable, example, configuration surface and
published documentation page. Cover `docs/`, root/package/example READMEs,
reader-facing help and the selected Book content used by assisted generation.
Keep historical delivery records historical; reconcile current product claims
against source rather than rewriting previously recorded validation results.

The inventory includes:

- application lifecycle, HTTP/web boundaries, middleware, rendering, HTMX and
  presentation helpers;
- configuration/settings, database, model, validation, seed and logging;
- authentication, sessions, authenticators, recovery and cryptographic support;
- mail, images, pubsub, scheduling, telemetry and testing helpers;
- generator commands, conversational/TUI flows, planning, execution, generated
  projects and managed documentation.

Each public surface has a row identifying its source owner, current documentation,
intended reader need, responsible slice, verification and outcome. Account for
every tracked documentation page as well as every discovered public surface.
Do not use a hand-picked list to exclude newly discovered packages or commands.

## Four Quadrants

Preserve `docs/README.md`, the four existing quadrant directories and README
directory entrypoints. Keep the User Guide at `docs/tutorials/user-guide/`.
Each page has one primary reader purpose and is reachable from its quadrant.
Connect supporting pages rather than duplicating the same prose in all four
quadrants. A surface may share a learning chapter or explanation with related
surfaces; reference coverage still accounts for its public contract.

## Source and Evidence

Use exported code, current executable help, effective configuration validation,
schemas and tested example composition as authority. Approved specifications
describe intent; they cannot establish that a behavior is delivered.

Record the inspected commit and relevant source paths for every coverage row.
Classify code blocks as executable examples or explicit contextual fragments.
Execute published commands in isolated owned fixtures. Compile Go examples and
check configuration against actual validators. Validate contextual fragments in
their stated composition instead of presenting them as standalone programs.

Evidence records exact commands, expected observable results, exit status and
safe diagnostic references. Empty test selectors, compilation-only checks and
simulated adapters cannot establish a successful real workflow. Report those
boundaries precisely. No live production credentials or external mail recipients
are needed for documentation acceptance.

## Correction Boundary

Correct prose, navigation, source comments, documentation examples and their
validation tooling. Preserve current runtime contracts. If an advertised journey
fails because of a runtime defect, record the defect through the repository
ticket/spec channel and block that coverage item. Do not silently change runtime
behavior or declare the documented successful outcome verified.

Generated documentation examples must conform to the current boxed contract.
This set verifies that contract's delivered behavior; it does not widen the
generator's capabilities to satisfy a documentation example.

## Independent Delivery

The delivery set is `documentation-conformance`, owned entirely by this
repository. Its specification, plan, tracker, branches, PRs, reports, fixture
storage, controller configuration, checkpoint and final gate are local to it.
Approval and progress are determined only by its own recorded artifacts.
One slice-loop execution integrates its approved slices into `dev` through
Rebase + Fast-forward. No executable or mutable state from a consumer project
is an execution prerequisite.

## Acceptance Criteria

| ID | Required outcome |
| --- | --- |
| DC01 | A reproducible inventory accounts for every public surface and every tracked product-documentation page at an identified source revision. |
| DC02 | All four quadrants and the User Guide have coherent intent, navigation, local links and valid heading anchors. |
| DC03 | Runtime, HTTP and presentation documentation matches current public APIs and observable behavior. |
| DC04 | Data, configuration and settings documentation matches effective validators, persistence contracts and examples. |
| DC05 | Authentication and crypto claims match current proof, freshness, ownership, recovery, error and secret-handling contracts. |
| DC06 | Infrastructure and helper documentation matches actual mail/image/event/job/telemetry/testing capabilities. |
| DC07 | Generator, CLI/TUI, Book and generated-project documentation matches current delivered commands and conformance. |
| DC08 | Every runnable example has successful execution evidence; contextual fragments have an explicit context and appropriate compilation/validation evidence. |
| DC09 | One clean immutable integrated dev candidate passes the repository-owned documentation gate; no unresolved or unverified coverage row is counted as current. |
| DC10 | Closure states the verified revision, full coverage and actual evidence limits, and leaves independent resumable controller/report state. |

## Completion Claim

The final claim is documentation conformance for the exact validated source
revision. It is not a permanent guarantee after code changes. Later product
changes must update their affected documentation and example evidence in the
same delivery. Completion is withheld for missing tools, failed journeys or
unresolved source/documentation disagreement.

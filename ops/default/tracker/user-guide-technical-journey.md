<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# User Guide Technical Journey Tracker

Status: Delivered
Delivery set: user-guide-technical-journey
Plan: `ops/default/plan/user-guide-technical-journey.md`
Ticket: `ops/default/ticket/solved/20260929090213-rebuild-user-guide-technical-journey.md`
Spec: none
Base branch: `dev`
Planning base: `4f783c75d9e9417d35ce4cc2e8a4118a10a89c19`
Active slice: none
Active tasks: none
Execution gate: delivered on `dev` at
`09246fedaee1a8c00f36b74dfe6eb5f7850edf04`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | journey foundation | delivered | `docs/user-guide-journey-foundation` | `docs(slice-1): establish the User Guide journey` | #48 | `ops/default/report/slices/user-guide-technical-journey/slice-1-journey-foundation.md` |
| Slice 2 | web interaction | delivered | `docs/user-guide-web-interaction` | `docs(slice-2): explain the Hatmax web interaction model` | #49 | `ops/default/report/slices/user-guide-technical-journey/slice-2-web-interaction.md` |
| Slice 3 | feature anatomy | delivered | `docs/user-guide-feature-anatomy` | `docs(slice-3): establish canonical feature anatomy` | #50 | `ops/default/report/slices/user-guide-technical-journey/slice-3-feature-anatomy.md` |
| Slice 4 | data lifecycle | delivered | `docs/user-guide-data-lifecycle` | `docs(slice-4): explain the Hatmax data lifecycle` | #51 | `ops/default/report/slices/user-guide-technical-journey/slice-4-data-lifecycle.md` |
| Slice 5 | identity and runtime | delivered | `docs/user-guide-identity-runtime` | `docs(slice-5): explain identity and runtime configuration` | #52 | `ops/default/report/slices/user-guide-technical-journey/slice-5-identity-runtime.md` |
| Slice 6 | application services | delivered | `docs/user-guide-application-services` | `docs(slice-6): explain Hatmax application services` | #53 | `ops/default/report/slices/user-guide-technical-journey/slice-6-application-services.md` |
| Slice 7 | testing and evolution | delivered | `docs/user-guide-testing-evolution` | `docs(slice-7): complete the User Guide journey` | #54 | `ops/default/report/slices/user-guide-technical-journey/slice-7-testing-evolution.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `docs(guide): define the technical journey` | `9605605` | `make docs-check` passed |
| T1.2 | complete | `docs(guide): explain application anatomy` | `b8fecef` | `make docs-check` passed |
| T1.3 | complete | `docs(guide): integrate lifecycle and wiring` | `92e7ee6` | `make docs-check` passed |
| T1.4 | complete | `docs(guide): retire superseded foundation pages` | `4dcd0c6` | link audit and `make docs-check` passed |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | complete | `docs(guide): explain requests and partials` | `24bae79` | `make docs-check` passed |
| T2.2 | complete | `docs(guide): explain forms and validation` | `0500f9d` | `make docs-check` passed |
| T2.3 | complete | `docs(guide): explain presentation primitives` | `949bef8` | `make docs-check` passed |
| T2.4 | complete | `docs(guide): retire superseded interaction pages` | `ea89fa9` | link audit and `make docs-check` passed |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | complete | `docs(guide): define canonical feature anatomy` | `ebcfa57` | `make docs-check` passed |
| T3.2 | complete | `docs(guide): connect feature boundaries` | `f7317d4` | `make docs-check` passed |
| T3.3 | complete | `docs(guide): reconcile feature examples` | `9d142e7` | example, Book, and reference audit passed |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | complete | `docs(guide): explain persistence and migrations` | `efe9828` | `make docs-check` passed |
| T4.2 | complete | `docs(guide): explain models and data flow` | `a7a8a4b` | `make docs-check` passed |
| T4.3 | complete | `docs(guide): retire superseded data pages` | `214cf19` | link audit and `make docs-check` passed |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | complete | `docs(guide): explain identity and sessions` | `e2702dd` | `make docs-check` passed |
| T5.2 | complete | `docs(guide): explain runtime configuration` | `ff3072a` | `make docs-check` passed |
| T5.3 | complete | `docs(guide): retire superseded runtime pages` | `b40e6e7` | link audit and `make docs-check` passed |

## Slice 6 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T6.1 | complete | `docs(guide): explain events and background work` | `d313be8` | `make docs-check` passed |
| T6.2 | complete | `docs(guide): explain application services` | `3859877` | `make docs-check` passed |
| T6.3 | complete | `docs(guide): connect services to application wiring` | `588547e` | link audit and `make docs-check` passed |

## Slice 7 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T7.1 | complete | `docs(guide): explain testing and evolution` | `3460cca` | `make docs-check` passed |
| T7.2 | complete | `docs(guide): frame experimental generation` | `49e7dec` | boundary and link audit passed |
| T7.3 | complete | `docs(guide): close the technical journey` | `6cf4ce8` | primitive coverage and navigation audit passed |

## Completion Gates

- [x] The maintainer approves the plan and tracker.
- [x] Slice 1 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 2 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 3 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 4 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 5 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 6 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] Slice 7 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] The exact integrated `dev` commit passes `make docs-check`.
- [x] The final report records a front-to-back editorial read and complete
  primitive-coverage audit.

## Current Gate

The seven slices are merged into `dev`. The exact integrated commit passed the
documentation gate, front-to-back editorial read, navigation audit, and public
package coverage audit. The delivery set is closed.

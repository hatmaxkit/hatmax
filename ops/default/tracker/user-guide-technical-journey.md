# User Guide Technical Journey Tracker

Status: Approved
Delivery set: user-guide-technical-journey
Plan: `ops/default/plan/user-guide-technical-journey.md`
Ticket: `ops/default/ticket/reviewing/20260929090213-rebuild-user-guide-technical-journey.md`
Spec: none
Base branch: `dev`
Planning base: `4f783c75d9e9417d35ce4cc2e8a4118a10a89c19`
Active slice: Slice 6
Active tasks: T6.1, T6.2, T6.3
Execution gate: satisfied when this approved activation is committed to `dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | journey foundation | delivered | `docs/user-guide-journey-foundation` | `docs(slice-1): establish the User Guide journey` | #48 | `ops/default/report/slices/user-guide-technical-journey/slice-1-journey-foundation.md` |
| Slice 2 | web interaction | delivered | `docs/user-guide-web-interaction` | `docs(slice-2): explain the Hatmax web interaction model` | #49 | `ops/default/report/slices/user-guide-technical-journey/slice-2-web-interaction.md` |
| Slice 3 | feature anatomy | delivered | `docs/user-guide-feature-anatomy` | `docs(slice-3): establish canonical feature anatomy` | #50 | `ops/default/report/slices/user-guide-technical-journey/slice-3-feature-anatomy.md` |
| Slice 4 | data lifecycle | delivered | `docs/user-guide-data-lifecycle` | `docs(slice-4): explain the Hatmax data lifecycle` | #51 | `ops/default/report/slices/user-guide-technical-journey/slice-4-data-lifecycle.md` |
| Slice 5 | identity and runtime | delivered | `docs/user-guide-identity-runtime` | `docs(slice-5): explain identity and runtime configuration` | #52 | `ops/default/report/slices/user-guide-technical-journey/slice-5-identity-runtime.md` |
| Slice 6 | application services | active | `docs/user-guide-application-services` | `docs(slice-6): explain Hatmax application services` |  | `ops/default/report/slices/user-guide-technical-journey/slice-6-application-services.md` |
| Slice 7 | testing and evolution | planned | `docs/user-guide-testing-evolution` | `docs(slice-7): complete the User Guide journey` |  | `ops/default/report/slices/user-guide-technical-journey/slice-7-testing-evolution.md` |

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
| T6.1 | pending | `docs(guide): explain events and background work` |  |  |
| T6.2 | pending | `docs(guide): explain application services` |  |  |
| T6.3 | pending | `docs(guide): connect services to application wiring` |  |  |

## Slice 7 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T7.1 | pending | `docs(guide): explain testing and evolution` |  |  |
| T7.2 | pending | `docs(guide): frame experimental generation` |  |  |
| T7.3 | pending | `docs(guide): close the technical journey` |  |  |

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
- [ ] Slice 6 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 7 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] The exact integrated `dev` commit passes `make docs-check`.
- [ ] The final report records a front-to-back editorial read and complete
  primitive-coverage audit.

## Current Gate

Slice 5 merged through pull request #52 at
`d1bf11a5c750dccb6954a54700c5e01bb2634a92`. Slice 6 is active and may begin
from that integrated `dev` state.

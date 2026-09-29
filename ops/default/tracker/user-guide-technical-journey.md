# User Guide Technical Journey Tracker

Status: Approved
Delivery set: user-guide-technical-journey
Plan: `ops/default/plan/user-guide-technical-journey.md`
Ticket: `ops/default/ticket/reviewing/20260929090213-rebuild-user-guide-technical-journey.md`
Spec: none
Base branch: `dev`
Planning base: `4f783c75d9e9417d35ce4cc2e8a4118a10a89c19`
Active slice: Slice 2
Active tasks: T2.1, T2.2, T2.3, T2.4
Execution gate: satisfied when this approved activation is committed to `dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | journey foundation | delivered | `docs/user-guide-journey-foundation` | `docs(slice-1): establish the User Guide journey` | #48 | `ops/default/report/slices/user-guide-technical-journey/slice-1-journey-foundation.md` |
| Slice 2 | web interaction | reviewing | `docs/user-guide-web-interaction` | `docs(slice-2): explain the Hatmax web interaction model` |  | `ops/default/report/slices/user-guide-technical-journey/slice-2-web-interaction.md` |
| Slice 3 | feature anatomy | planned | `docs/user-guide-feature-anatomy` | `docs(slice-3): establish canonical feature anatomy` |  | `ops/default/report/slices/user-guide-technical-journey/slice-3-feature-anatomy.md` |
| Slice 4 | data lifecycle | planned | `docs/user-guide-data-lifecycle` | `docs(slice-4): explain the Hatmax data lifecycle` |  | `ops/default/report/slices/user-guide-technical-journey/slice-4-data-lifecycle.md` |
| Slice 5 | identity and runtime | planned | `docs/user-guide-identity-runtime` | `docs(slice-5): explain identity and runtime configuration` |  | `ops/default/report/slices/user-guide-technical-journey/slice-5-identity-runtime.md` |
| Slice 6 | application services | planned | `docs/user-guide-application-services` | `docs(slice-6): explain Hatmax application services` |  | `ops/default/report/slices/user-guide-technical-journey/slice-6-application-services.md` |
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
| T3.1 | pending | `docs(guide): define canonical feature anatomy` |  |  |
| T3.2 | pending | `docs(guide): connect feature boundaries` |  |  |
| T3.3 | pending | `docs(guide): reconcile feature examples` |  |  |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | pending | `docs(guide): explain persistence and migrations` |  |  |
| T4.2 | pending | `docs(guide): explain models and data flow` |  |  |
| T4.3 | pending | `docs(guide): retire superseded data pages` |  |  |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | pending | `docs(guide): explain identity and sessions` |  |  |
| T5.2 | pending | `docs(guide): explain runtime configuration` |  |  |
| T5.3 | pending | `docs(guide): retire superseded runtime pages` |  |  |

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
- [ ] Slice 2 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 3 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 4 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 5 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 6 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] Slice 7 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] The exact integrated `dev` commit passes `make docs-check`.
- [ ] The final report records a front-to-back editorial read and complete
  primitive-coverage audit.

## Current Gate

Slice 2 content and focused validation are complete. Its report is in
reviewing state; the recorded pull request must be opened against `dev`.

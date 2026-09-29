# User Guide Technical Journey Tracker

Status: Proposed
Delivery set: user-guide-technical-journey
Plan: `ops/default/plan/user-guide-technical-journey.md`
Ticket: `ops/default/ticket/open/20260929090213-rebuild-user-guide-technical-journey.md`
Spec: none
Base branch: `dev`
Planning base: `4f783c75d9e9417d35ce4cc2e8a4118a10a89c19`
Active slice: none
Active tasks: none
Execution gate: awaiting maintainer approval

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | journey foundation | proposed | `docs/user-guide-journey-foundation` | `docs(slice-1): establish the User Guide journey` |  | `ops/default/report/slices/user-guide-technical-journey/slice-1-journey-foundation.md` |
| Slice 2 | web interaction | proposed | `docs/user-guide-web-interaction` | `docs(slice-2): explain the Hatmax web interaction model` |  | `ops/default/report/slices/user-guide-technical-journey/slice-2-web-interaction.md` |
| Slice 3 | feature anatomy | proposed | `docs/user-guide-feature-anatomy` | `docs(slice-3): establish canonical feature anatomy` |  | `ops/default/report/slices/user-guide-technical-journey/slice-3-feature-anatomy.md` |
| Slice 4 | data lifecycle | proposed | `docs/user-guide-data-lifecycle` | `docs(slice-4): explain the Hatmax data lifecycle` |  | `ops/default/report/slices/user-guide-technical-journey/slice-4-data-lifecycle.md` |
| Slice 5 | identity and runtime | proposed | `docs/user-guide-identity-runtime` | `docs(slice-5): explain identity and runtime configuration` |  | `ops/default/report/slices/user-guide-technical-journey/slice-5-identity-runtime.md` |
| Slice 6 | application services | proposed | `docs/user-guide-application-services` | `docs(slice-6): explain Hatmax application services` |  | `ops/default/report/slices/user-guide-technical-journey/slice-6-application-services.md` |
| Slice 7 | testing and evolution | proposed | `docs/user-guide-testing-evolution` | `docs(slice-7): complete the User Guide journey` |  | `ops/default/report/slices/user-guide-technical-journey/slice-7-testing-evolution.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | proposed | `docs(guide): define the technical journey` |  |  |
| T1.2 | proposed | `docs(guide): explain application anatomy` |  |  |
| T1.3 | proposed | `docs(guide): integrate lifecycle and wiring` |  |  |
| T1.4 | proposed | `docs(guide): retire superseded foundation pages` |  |  |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | proposed | `docs(guide): explain requests and partials` |  |  |
| T2.2 | proposed | `docs(guide): explain forms and validation` |  |  |
| T2.3 | proposed | `docs(guide): explain presentation primitives` |  |  |
| T2.4 | proposed | `docs(guide): retire superseded interaction pages` |  |  |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | proposed | `docs(guide): define canonical feature anatomy` |  |  |
| T3.2 | proposed | `docs(guide): connect feature boundaries` |  |  |
| T3.3 | proposed | `docs(guide): reconcile feature examples` |  |  |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | proposed | `docs(guide): explain persistence and migrations` |  |  |
| T4.2 | proposed | `docs(guide): explain models and data flow` |  |  |
| T4.3 | proposed | `docs(guide): retire superseded data pages` |  |  |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | proposed | `docs(guide): explain identity and sessions` |  |  |
| T5.2 | proposed | `docs(guide): explain runtime configuration` |  |  |
| T5.3 | proposed | `docs(guide): retire superseded runtime pages` |  |  |

## Slice 6 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T6.1 | proposed | `docs(guide): explain events and background work` |  |  |
| T6.2 | proposed | `docs(guide): explain application services` |  |  |
| T6.3 | proposed | `docs(guide): connect services to application wiring` |  |  |

## Slice 7 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T7.1 | proposed | `docs(guide): explain testing and evolution` |  |  |
| T7.2 | proposed | `docs(guide): frame experimental generation` |  |  |
| T7.3 | proposed | `docs(guide): close the technical journey` |  |  |

## Completion Gates

- [ ] The maintainer approves the plan and tracker.
- [ ] Slice 1 is delivered through its recorded branch, report, pull request,
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

Planning is proposed. No slice is active, and documentation implementation is
not authorized until the maintainer approves the plan and tracker.

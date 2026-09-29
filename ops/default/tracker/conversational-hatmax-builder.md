# Conversational Hatmax Builder Tracker

Status: Approved
Delivery set: conversational-hatmax-builder
Plan: `ops/default/plan/conversational-hatmax-builder.md`
Umbrella spec: `ops/default/spec/interactive-hatmax-generator.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`
Approved specs:

- `ops/default/spec/canonical-application-scaffold.md`
- `ops/default/spec/conversational-hatmax-surface.md`
- `ops/default/spec/conversational-hatmax-surface-model.md`

Base branch: `dev`
Planning base: `510a856419a97d64959387099c48a81f867fb09a`
Active slice: Slice 4
Active tasks: T4.1, T4.2, T4.3
Execution gate: satisfied by `cbea3e0838c4ef25229b3822aebcf0f1dbcd2a91`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Builder contracts | delivered | `docs/hatmax-builder-contracts` | `docs(slice-1): define Hatmax builder contracts` | #56 | `ops/default/report/slices/conversational-hatmax-builder/slice-1-builder-contracts.md` |
| Slice 2 | Application planning kernel | delivered | `feat/application-scaffold-kernel` | `feat(slice-2): plan canonical Hatmax applications` | #57 | `ops/default/report/slices/conversational-hatmax-builder/slice-2-application-planning-kernel.md` |
| Slice 3 | Scaffold execution | delivered | `feat/application-scaffold-rendering` | `feat(slice-3): render canonical Hatmax applications` | #58 | `ops/default/report/slices/conversational-hatmax-builder/slice-3-scaffold-execution.md` |
| Slice 4 | Application bootstrap product | active | `feat/application-scaffold-product` | `feat(slice-4): deliver Hatmax application bootstrap` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-4-application-bootstrap-product.md` |
| Slice 5 | Conversation state | planned | `feat/conversational-state` | `feat(slice-5): persist Hatmax conversations` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-5-conversation-state.md` |
| Slice 6 | Conversational coordinator | planned | `feat/conversational-coordinator` | `feat(slice-6): coordinate conversational Hatmax work` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-6-conversational-coordinator.md` |
| Slice 7 | Hatmax TUI | planned | `feat/hatmax-tui` | `feat(slice-7): deliver the conversational Hatmax TUI` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-7-hatmax-tui.md` |
| Slice 8 | Builder acceptance | planned | `test/conversational-builder-acceptance` | `test(slice-8): validate the conversational Hatmax builder` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-8-builder-acceptance.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `docs(spec): reconcile application scaffold contracts` | `0d2748bcffcc7f9334dc8e322444610cdb21efe8` | `make docs-check`; `git diff --check` passed |
| T1.2 | complete | `docs(spec): reconcile conversational surface contracts` | `d6a5b6a9db9376bd8bcacf08ddce92bbbcfb26e7` | `make docs-check`; `git diff --check` passed |
| T1.3 | complete | `docs(spec): approve the conversational Hatmax builder` | `d040d281781bdef13b42c320d2c8b1a3871aa379` | `make docs-check`; `git diff --check` passed |

## Slice 2 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T2.1 | complete | `feat(generator): inventory application targets` | `606e132863d39492c8f49481eb3212200d1baf5a` | target inventory tests and full slice gate passed |
| T2.2 | complete | `feat(generator): define application intents` | `08322e5d2ea663184d89f1725bb94ac9b1307405` | normalization, schema, and semantic validation tests; full slice gate passed |
| T2.3 | complete | `feat(generator): plan application scaffolds` | `347dca571e118df7b581c13a912cece2cd3e1bb3` | Book, composite plan, serialization, digest, preservation, and drift tests; full slice gate passed |

## Slice 3 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T3.1 | complete | `feat(generator): render Hatmax application foundations` | `835e849676828f87cbf75445ad4584607a0466aa` | foundation recipes, thin-main contract, generated compilation, and full slice gate passed |
| T3.2 | complete | `feat(generator): render Hatmax web scaffolds` | `0a62b3b67f1c38acbb60c9c50490a78eb291b499` | router, middleware, templates, assets, neutral-page tests, and full slice gate passed |
| T3.3 | complete | `test(generator): enforce scaffold execution` | `e6fdaffd0389f0ea19a8d65d839c128789de8d2e` | atomic publication, preservation, rollback, drift, conformance, compilation, test classification, and full slice gate passed |

## Slice 4 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T4.1 | pending | `feat(generator): interpret application creation` | pending | schema, Codex context, clarification, and isolation tests; full slice gate pending |
| T4.2 | pending | `feat(generator): coordinate application bootstrap` | pending | approval, composite units, execution, conformance, and reporting tests; full slice gate pending |
| T4.3 | pending | `test(generator): validate application bootstrap` | pending | deterministic and terminal scaffold acceptance; full slice gate pending |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | pending | `feat(generator): define conversational results` | pending | dialogue schema, bounds, and mutation isolation tests; full slice gate pending |
| T5.2 | pending | `feat(generator): model persistent conversations` | pending | lifecycle, stale, reset, rebind, and no-approval tests; full slice gate pending |
| T5.3 | pending | `feat(generator): persist local conversation state` | pending | migration, atomicity, privacy, restart, and pruning tests; full slice gate pending |

## Slice 6 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T6.1 | pending | `feat(generator): coordinate conversational turns` | pending | dialogue, candidate, clarification, plan-ready, and unsupported tests; full slice gate pending |
| T6.2 | pending | `feat(generator): resume Hatmax conversations` | pending | resume, reinspection, rebind, reset, and stale tests; full slice gate pending |
| T6.3 | pending | `feat(generator): recover conversational operations` | pending | cancellation, rollback, retained changes, invalid approval, and retry tests; full slice gate pending |

## Slice 7 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T7.1 | pending | `feat(cli): add the hm command` | pending | TUI lifecycle, input, resize, cancellation, and restoration tests; full slice gate pending |
| T7.2 | pending | `feat(tui): present conversational Hatmax work` | pending | conversation, plan, progress, failure, and accessibility tests; full slice gate pending |
| T7.3 | pending | `feat(cli): preserve Hatmax headless compatibility` | pending | parity, alias, exit status, and routing tests; full slice gate pending |

## Slice 8 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T8.1 | pending | `test(generator): exercise conversational workflows` | pending | scaffold, resume, evolution, reset, drift, cancellation, recovery, and rejection acceptance; full slice gate pending |
| T8.2 | pending | `test(generator): validate live Hatmax conversations` | pending | authenticated backend, thread reuse, rebind, and isolation smoke; full slice gate pending |
| T8.3 | pending | `docs(generator): document conversational Hatmax` | pending | reference, User Guide, migration, examples, changelog, and docs gate; full slice gate pending |

## Completion Gates

- [x] The maintainer approves the plan and tracker.
- [x] Slice 1 promotes one reconciled non-conflicting specification set.
- [x] Slice 1 is delivered through its branch, report, pull request, and merge.
- [x] Slice 2 is delivered through its branch, report, pull request, and merge.
- [x] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 5 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 6 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 7 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 8 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes the delivery-set gate.

## Current Gate

Slice 3 was delivered by #58. Slice 4 is approved for execution on
`feat/application-scaffold-product`; T4.1, T4.2, and T4.3 are active.

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
Active slice: Slice 7
Active tasks: T7.1, T7.2, T7.3
Execution gate: satisfied by `cbea3e0838c4ef25229b3822aebcf0f1dbcd2a91`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Builder contracts | delivered | `docs/hatmax-builder-contracts` | `docs(slice-1): define Hatmax builder contracts` | #56 | `ops/default/report/slices/conversational-hatmax-builder/slice-1-builder-contracts.md` |
| Slice 2 | Application planning kernel | delivered | `feat/application-scaffold-kernel` | `feat(slice-2): plan canonical Hatmax applications` | #57 | `ops/default/report/slices/conversational-hatmax-builder/slice-2-application-planning-kernel.md` |
| Slice 3 | Scaffold execution | delivered | `feat/application-scaffold-rendering` | `feat(slice-3): render canonical Hatmax applications` | #58 | `ops/default/report/slices/conversational-hatmax-builder/slice-3-scaffold-execution.md` |
| Slice 4 | Application bootstrap product | delivered | `feat/application-scaffold-product` | `feat(slice-4): deliver Hatmax application bootstrap` | #59 | `ops/default/report/slices/conversational-hatmax-builder/slice-4-application-bootstrap-product.md` |
| Slice 5 | Conversation state | delivered | `feat/conversational-state` | `feat(slice-5): persist Hatmax conversations` | #60 | `ops/default/report/slices/conversational-hatmax-builder/slice-5-conversation-state.md` |
| Slice 6 | Conversational coordinator | delivered | `feat/conversational-coordinator` | `feat(slice-6): coordinate conversational Hatmax work` | #61 | `ops/default/report/slices/conversational-hatmax-builder/slice-6-conversational-coordinator.md` |
| Slice 7 | Hatmax TUI | reviewing | `feat/hatmax-tui` | `feat(slice-7): deliver the conversational Hatmax TUI` | #62 | `ops/default/report/slices/conversational-hatmax-builder/slice-7-hatmax-tui.md` |
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
| T4.1 | complete | `feat(generator): interpret application creation` | `a13eda473093434b51e5bde79f1fd636cd392156` | schema, Codex context, clarification, isolation tests, and full slice gate passed |
| T4.2 | complete | `feat(generator): coordinate application bootstrap` | `098a024899b43982973c75df2774a6b35e3f2865` | approval, composite units, execution, conformance, reporting tests, and full slice gate passed |
| T4.3 | complete | `test(generator): validate application bootstrap` | `e17263f0cfae8aa97ba05104680c319a14abd9ec` | deterministic and terminal scaffold acceptance, real composite compilation, and full slice gate passed |

## Slice 5 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T5.1 | complete | `feat(generator): define conversational results` | `46f6fae36f0f9c6816547c3fb6f26eb909ad14a7` | dialogue schema, bounds, mutation isolation tests, and full slice gate passed |
| T5.2 | complete | `feat(generator): model persistent conversations` | `06f828b8b3a826c5a441154d60f9d4629c982e39` | lifecycle, stale, reset, rebind, no-approval tests, and full slice gate passed |
| T5.3 | complete | `feat(generator): persist local conversation state` | `f3d6fab4a42d863da2d7689c7786417f7343c818` | incompatible schema, atomicity, privacy, restart, lock, rebinding, pruning tests, and full slice gate passed |

## Slice 6 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T6.1 | complete | `feat(generator): coordinate conversational turns` | `5c99651dbf212c1df63c17854f5655e8d66e1c2e` | dialogue, candidate, clarification, plan-ready, unsupported, and digest-bound approval tests; full slice gate passed |
| T6.2 | complete | `feat(generator): resume Hatmax conversations` | `ff8f3dff9392b67513628c8befffc37fd3d09ec5` | resume, reinspection, rebind, reset, thread replacement, and stale tests; full slice gate passed |
| T6.3 | complete | `feat(generator): recover conversational operations` | `4813733e113dc8725221e69eb3113615e8214dba` | cancellation, persistence failure, retained changes, invalid approval, incomplete validation, and safe retry tests; full slice gate passed |

## Slice 7 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T7.1 | complete | `feat(cli): add the hm command` | `113b7ae65a872f195e990233412a299bc1c1a412` | TUI lifecycle, input, resize, cancellation, and restoration tests; full slice gate passed |
| T7.2 | complete | `feat(tui): present conversational Hatmax work` | `2310853a755729b475004618ded323ec1ebb7db6` | conversation, plan, progress, failure, and accessibility tests; full slice gate passed |
| T7.3 | complete | `feat(cli): preserve Hatmax headless compatibility` | `a38960c0c6f503038c43bd6288648b248316047b` | parity, alias, exit status, and routing tests; full slice gate passed |

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
- [x] Slice 4 is delivered through its branch, report, pull request, and merge.
- [x] Slice 5 is delivered through its branch, report, pull request, and merge.
- [x] Slice 6 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 7 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 8 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes the delivery-set gate.

## Current Gate

Slice 7 implementation and validation are complete on `feat/hatmax-tui`;
maintainer review and merge are pending.

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
Active slice: Slice 2
Active tasks: none; slice review pending
Execution gate: satisfied by `cbea3e0838c4ef25229b3822aebcf0f1dbcd2a91`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Builder contracts | delivered | `docs/hatmax-builder-contracts` | `docs(slice-1): define Hatmax builder contracts` | #56 | `ops/default/report/slices/conversational-hatmax-builder/slice-1-builder-contracts.md` |
| Slice 2 | Application planning kernel | reviewing | `feat/application-scaffold-kernel` | `feat(slice-2): plan canonical Hatmax applications` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-2-application-planning-kernel.md` |
| Slice 3 | Scaffold execution | planned | `feat/application-scaffold-rendering` | `feat(slice-3): render canonical Hatmax applications` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-3-scaffold-execution.md` |
| Slice 4 | Application bootstrap product | planned | `feat/application-scaffold-product` | `feat(slice-4): deliver Hatmax application bootstrap` | pending | `ops/default/report/slices/conversational-hatmax-builder/slice-4-application-bootstrap-product.md` |
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
| T3.1 | pending | `feat(generator): render Hatmax application foundations` | pending | foundation renderer and generated compile tests; full slice gate pending |
| T3.2 | pending | `feat(generator): render Hatmax web scaffolds` | pending | web, template, asset, and no-domain tests; full slice gate pending |
| T3.3 | pending | `test(generator): enforce scaffold execution` | pending | atomicity, collision, conformance, compile, and validation classification; full slice gate pending |

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
- [ ] Slice 2 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 3 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 4 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 5 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 6 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 7 is delivered through its branch, report, pull request, and merge.
- [ ] Slice 8 is delivered through its branch, report, pull request, and merge.
- [ ] The exact integrated `dev` candidate passes the delivery-set gate.

## Current Gate

Slice 1 was delivered by #56. Slice 2 is approved for execution on
`feat/application-scaffold-kernel`; T2.1, T2.2, and T2.3 are active.

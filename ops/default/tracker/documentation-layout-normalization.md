# Documentation Layout Normalization Tracker

Status: Approved
Delivery set: documentation-layout-normalization
Plan: `ops/default/plan/documentation-layout-normalization.md`
Ticket: `ops/default/ticket/ready/20260929091830-normalize-documentation-entrypoints-and-media.md`
Spec: `ops/default/spec/boxed-diataxis-documentation.md`
Base branch: `dev`
Planning base: `73fcc8c076b30259b522da8fbb056bf6ae123da1`
Active slice: Slice 1
Active tasks: T1.1, T1.2, T1.3, T1.4
Execution gate: satisfied when this approved tracker and plan are committed to
`dev`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | canonical README layout | ready | `feat/documentation-readme-layout` | `feat(slice-1): adopt forge-native documentation layout` |  | `ops/default/report/slices/documentation-layout-normalization/slice-1-canonical-readme-layout.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | pending | `docs: adopt forge-native documentation entrypoints` |  | `make docs-check`; link and layout audit |
| T1.2 | pending | `docs(generator): adopt the README documentation contract` |  | accepted-spec and Book-rule audit |
| T1.3 | pending | `feat(generator): render README documentation entrypoints` |  | `go test ./generator/... ./internal/hatmaxcli/...`; `make lint-strict` |
| T1.4 | pending | `docs(ops): report documentation layout normalization` |  | `git diff --check` |

## Completion Gates

- [x] The maintainer approved the layout decision and delivery order.
- [x] The plan and tracker define one atomic slice.
- [ ] The plan and tracker are committed on `dev`.
- [ ] Slice 1 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] The exact integrated `dev` commit passes `make check`.
- [ ] Forgejo renders directory `README.md` files as documentation landing
  pages.
- [ ] The ticket is solved with its implementation commits recorded.

## Current Gate

The approved planning artifacts must be committed on `dev` before the slice
branch is created.

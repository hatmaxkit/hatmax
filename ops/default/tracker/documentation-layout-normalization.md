# Documentation Layout Normalization Tracker

Status: Approved
Delivery set: documentation-layout-normalization
Plan: `ops/default/plan/documentation-layout-normalization.md`
Ticket: `ops/default/ticket/reviewing/20260929091830-normalize-documentation-entrypoints-and-media.md`
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
| Slice 1 | canonical README layout | reviewing | `feat/documentation-readme-layout` | `feat(slice-1): adopt forge-native documentation layout` | #47 | `ops/default/report/slices/documentation-layout-normalization/slice-1-canonical-readme-layout.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `docs: adopt forge-native documentation entrypoints` | `a854c27` | `make docs-check` passed; link and layout audit passed |
| T1.2 | complete | `docs(generator): adopt the README documentation contract` | `92320c0` | accepted-spec and Book-rule audit passed |
| T1.3 | complete | `feat(generator): render README documentation entrypoints` | `6ba9c6b` | `go test ./generator/... ./internal/hatmaxcli/...` passed; `make lint-strict` passed |
| T1.4 | complete | `docs(ops): report documentation layout normalization` | `b6225d7` | `git diff --check` passed |

## Completion Gates

- [x] The maintainer approved the layout decision and delivery order.
- [x] The plan and tracker define one atomic slice.
- [x] The plan and tracker are committed on `dev`.
- [ ] Slice 1 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [ ] The exact integrated `dev` commit passes `make check`.
- [ ] Forgejo renders directory `README.md` files as documentation landing
  pages.
- [ ] The ticket is solved with its implementation commits recorded.

## Current Gate

Slice 1 implementation and focused validation are complete. The report is in
reviewing state; the recorded pull request must be opened against `dev`.

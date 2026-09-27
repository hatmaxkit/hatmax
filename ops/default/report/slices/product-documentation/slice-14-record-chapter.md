# Slice 14: Record Chapter

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-14-record-chapter`
PR: `#17`

## Purpose

Teach storing one note that is still listed after a restart, and list that
chapter from the User Guide.

## Delivered Behavior

A reader can open Save a Record from the User Guide index. The first start
lists `Welcome`. Saving `Hello` and starting the process again still lists
`Welcome` and `Hello`. `Welcome` is not inserted a second time.

## Implementation Notes

The welcome row is a `seed.Seeder` tracked in `_seeds`. Saved rows use
`model.NewID`, `model.Now`, and `slug.Generate`. The notes page is not behind
sign-in.

## Contracts Added or Changed

The Save a Record chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/records.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

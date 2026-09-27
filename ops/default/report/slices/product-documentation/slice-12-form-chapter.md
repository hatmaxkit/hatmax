# Slice 12: Form Chapter

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-12-form-chapter`
PR: `#15`

## Purpose

Teach one form that rejects invalid input and accepts valid input, and list
that chapter from the User Guide.

## Delivered Behavior

A reader can open Accept a Form from the User Guide index. A POST without
`Origin` receives `403`. An empty name produces `name: is required`. One
character produces `name: must be at least 2 characters`. `Ada` produces
`Accepted Ada`.

## Implementation Notes

The form tags come from `ui.NewForm`. No CSRF token is set. Same-origin
protection is `middleware.RequireSameOrigin`. Validation is
`validation.Field`. The name is not stored.

## Contracts Added or Changed

The Accept a Form chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/forms.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

# Slice 16: Settings Chapter

Status: reviewing
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-16-settings-chapter`
PR: pending

## Purpose

Teach one runtime setting that is distinct from static configuration, add the
wiring appendix, and close the User Guide.

## Delivered Behavior

A reader can open Change Settings at Runtime as chapter 8. The first read is
`Hello`. After a POST, the next read is `Hi`. A restart returns `Hello`, and
`config.yaml` is unchanged.

The User Guide index links all eight chapters in order. Appendices link
wiring, image storage, telemetry, crypto primitives, and the Ticked example.
`docs/index.md` leads with the User Guide.

## Implementation Notes

The sample store is in memory. The settings package does not provide a
Postgres store. The wiring page lists the setup order used in this guide and
links the swappable interfaces instead of copying them.

## Contracts Added or Changed

The settings chapter and the wiring appendix. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/settings.md`
- `docs/tutorials/user-guide/wiring.md`
- `docs/tutorials/user-guide/index.md`
- `docs/index.md`

## Validation

- `git diff --check` passed for the task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

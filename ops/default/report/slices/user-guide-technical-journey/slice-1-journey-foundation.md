# Slice 1: Journey Foundation

Status: reviewing
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-journey-foundation`
PR: #48

## Purpose

Replace the example-driven User Guide entry with the application model,
anatomy, and lifecycle concepts that readers need before learning individual
Hatmax capabilities.

## Delivered Behavior

The User Guide now declares a complete sixteen-chapter technical journey and
separates that journey from the future practical Todo tutorial. Orientation
defines Hatmax-owned capabilities, application-owned product behavior, and the
deliberate server-rendered, HTMX, Postgres-first constraints.

Application Anatomy presents the complete project shape, a `main.go`
composition root containing only `main`, feature ownership, adapter boundaries,
and an order for reading an unfamiliar project. Lifecycle and Wiring then
connects constructed values to ordered startup, delayed route registration,
rollback alignment, and reverse shutdown.

The superseded first-process exercise and wiring appendix were removed. Their
durable lifecycle content is present in the new foundation, while the focused
application bootstrap remains in the existing how-to guide.

## Implementation Notes

Later journey titles are visible from the start so readers can understand the
route from application model through assisted generation. Chapters not yet
delivered remain unlinked, avoiding placeholders and broken navigation.

The remaining current capability walkthroughs stay reachable until their
corresponding slices replace them. This keeps the documentation tree complete
after every independently reviewable slice.

## Contracts Added or Changed

- The User Guide teaches application composition rather than one cumulative
  repository exercise.
- `main.go` is documented as the visible composition root with one `main`
  function and no feature behavior.
- Wiring is part of the primary journey rather than an appendix.
- Lifecycle participation is limited to genuine startup, shutdown, and route
  responsibilities.
- The practical Todo build remains a separate future tutorial.

## Files of Interest

- `docs/tutorials/user-guide/README.md`
- `docs/tutorials/user-guide/orientation.md`
- `docs/tutorials/user-guide/application-anatomy.md`
- `docs/tutorials/user-guide/lifecycle-and-wiring.md`
- `docs/explanation/interfaces-and-adapters/README.md`

## Validation

- `make docs-check` passed.
- Foundation heading and reading-order audit passed.
- Superseded foundation-link audit passed.
- `git diff --check` passed.

## Risks and Follow-ups

The remaining capability walkthroughs still use the previous exercise style.
Slices 2 through 7 replace them in journey order; this slice does not rewrite
those subjects early.

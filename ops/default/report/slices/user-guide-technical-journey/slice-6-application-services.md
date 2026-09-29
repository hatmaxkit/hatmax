# Slice 6: Application Services

Status: reviewing
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-application-services`
PR: #53

## Purpose

Place events, scheduled work, mail, images, and telemetry in the Hatmax
application model with explicit interfaces, lifecycle, and wiring.

## Delivered Behavior

The guide now explains post-commit event publication, durable and in-memory
brokers, at-least-once subscribers, scheduler jobs and handlers, current retry
and recurrence limits, and dependency-order startup and shutdown.

Application services now cover mail provider selection and no-op semantics,
image metadata, storage and processing boundaries, and bounded telemetry. Each
adapter remains explicitly selected in `main` and consumed through a narrow
feature-owned interface.

The superseded background-work exercise was removed after its durable event,
subscriber identity, restart, and idempotency material was incorporated.

## Validation

- `make docs-check` passed.
- Capability audit passed for `pubsub`, `scheduler`, `mailer`, `image`, and
  `telemetry`.
- Current backend guarantees and limitations were checked against reference.
- Superseded `background-work.md` link audit passed.
- `git diff --check` passed.

## Risks and Follow-ups

Testing, operational evolution, assisted generation, and the final primitive
coverage audit remain in Slice 7.

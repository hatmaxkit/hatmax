# Slice 7: Testing and Evolution

Status: reviewing
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-testing-evolution`
PR: #54

## Purpose

Complete the User Guide with boundary-driven testing, cross-surface evolution,
and a brief factual orientation to experimental assisted generation.

## Delivered Behavior

The guide now connects model, service, handler, Postgres, adapter, template,
and composition tests to the contracts they prove. It explains focused and
repository gates, real Postgres requirements, project-owned acceptance, safe
schema evolution, and a final assembly review.

Assisted generation now appears only after the manual application model. The
chapter states that the product currently exposes a CLI, expects a future
conversational TUI, changes existing compatible applications, uses Codex only
for bounded interpretation, requires plan approval, and remains constrained
to the Hatmax Book. The prior command-driven generator exercise was removed.

The index now exposes one complete sixteen-chapter path with no transitional
walkthrough section. A package-map audit confirmed an intentional place for
every public Hatmax package and adapter family.

## Contracts Added or Changed

- Tests follow behavior boundaries rather than file count.
- Real Postgres tests own persistence claims; small consumer fakes own unit
  boundaries.
- Cross-surface changes enumerate obligations before editing.
- Assisted generation does not replace manual architecture knowledge or
  product review.
- Generator documentation remains explicit-only and boxed by Diataxis.
- The User Guide is one complete technical journey, distinct from the future
  practical Todo tutorial.

## Files of Interest

- `docs/tutorials/user-guide/testing-and-evolution.md`
- `docs/tutorials/user-guide/assisted-generation.md`
- `docs/tutorials/user-guide/README.md`
- `docs/reference/package-map/README.md`

## Validation

- `make docs-check` passed.
- All sixteen chapter titles and index links were audited.
- Every package in the public Package Map is covered or deliberately linked.
- Superseded User Guide page and inbound-link audit passed.
- Generator claims were checked against the current generator reference.
- `git diff --check` passed.

## Risks and Follow-ups

The separate practical Todo tutorial remains future work. The conversational
generator TUI remains exploratory and is not presented as delivered behavior.

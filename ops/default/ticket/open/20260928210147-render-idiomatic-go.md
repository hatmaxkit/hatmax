---
id: TKT-20260928210147
title: Render canonical idiomatic Go source
status: open
kind: bug
severity: high
priority: normal
scope: domain
tags: generator, rendering, go
source: review
reported_at: 2026-09-28T21:01:47Z
commits:
---

## Observed Behavior

The CRUD renderers emit Go that is syntactically formatted but does not follow
Hatmax's strict Go style. Generated services, handlers, stores, and tests use
adjacent control blocks without separating blank lines, compressed one-line
functions and structs, and returns attached to preceding statements. Other
non-canonical details include capitalized error strings and routes assembled
through constant string concatenation.

## Expected Outcome

Every canonical renderer must emit readable Go matching the conventions used
by Hatmax examples and passing the repository's strict lint policy without an
automatic repair step. Generated production code and generated tests are both
part of this contract.

## Validation

- Exercise every supported field kind and all three initial operations.
- Run `gofmt` and the strict Hatmax linters over the rendered corpus.
- Compare the rendered wiring, feature package, and tests with the canonical
  example patterns.
- Confirm repeated rendering remains deterministic.

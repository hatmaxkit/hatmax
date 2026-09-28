---
id: TKT-20260928210146
title: Validate generated projects with real build and lint gates
status: open
kind: bug
severity: high
priority: normal
scope: ops
tags: generator, acceptance, validation
source: manual_test
reported_at: 2026-09-28T21:01:46Z
commits:
---

## Observed Behavior

The terminal acceptance fixture supplies undefined `buildLogger`,
`buildDatabase`, `buildMigrator`, and `buildTemplates` helpers. Its injected
`make` and `sqlc` executables return success without compiling, generating, or
linting the resulting project. The acceptance suite can therefore report a
completed generation even when the retained project would fail its real
repository gates.

## Expected Outcome

Keep deterministic command doubles where failure injection requires them, but
add a representative, compilable Hatmax fixture whose generated output runs
real build, test, generation, formatting, and strict lint checks. Its
composition root must assemble real Hatmax dependencies inline and retain
only the `main` function.

## Validation

- Generate the canonical CRUD feature into the representative fixture.
- Run the fixture's real repository validation commands.
- Run strict `nlreturn`, `noinlineerr`, and `wsl_v5` linting over generated Go.
- Prove that an invalid composition root or uncompilable output fails the
  acceptance gate.

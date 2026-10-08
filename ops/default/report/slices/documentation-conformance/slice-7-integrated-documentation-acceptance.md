<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 7: Integrated Documentation Acceptance

Status: reviewing
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-documentation-acceptance`
PR: `#118`
Activation revision: `33ee2c28bc5d9756af78f8af2f543fd0f2b79deb`
Documentation task: `060c3fb18588b261f877848d0d73a7eb8bf1188f`
Validation task: `7236877162dacffefc1d063756ff8b9ab19e40b7`
Report introduction: `060c3fb18588b261f877848d0d73a7eb8bf1188f`

## Purpose

Reconcile the remaining entrypoints and complete source-bound coverage, then
bind integrated documentation acceptance to one clean immutable candidate and
actual execution with the required native toolchain.

## Delivered Behavior

The root quick start binds current source, resolves dependencies after imports
exist, uses the current caller-owned server API and coordinates shutdown. Source
installation replaces the unavailable published hm command. Existing-project
headless generation uses the reference request. Root guidance preserves the
known fresh-scaffold and field-update failures.

Documentation indexes now describe the User Guide's connected application model,
consistent with its chapters. They no longer promise a step-by-step project
tutorial. The package map includes generator, Book and command responsibilities.

## Implementation Notes

Slice 6 closed after verified canonical PR #117 integration and the controller's
exact-head evidence. Slice 7 preserves its approved branch, title and two tasks.
T7.1 reconciles the remaining 16 entrypoint rows and sweeps the 451-row inventory.
T7.2 binds 16 entrypoint rows: five executed root examples and eleven source
comparisons/navigation bindings. The root Go/YAML/module procedure matches the
executed bootstrap bytes. Its exact source-install/TUI procedure uses an owned
native PTY and the headless request matches the production CLI fixture request.
Three generator rows remain blocked by their recorded runtime defects.

The validator replaces its link extractor with Go. The gate resolves native
Go 1.27.1, pins local toolchain selection, records the finite executable set and
runs every cumulative workflow with inherited lookup directories removed.
Go's runner may prepend its own verified Go/formatter directory. The integrated
receipt binds the five owning execution receipts, native manifest, source/head,
compiler and diagnostic identities. Missing, stale, unbound or blocked evidence
cannot establish completed acceptance.

## Contracts Added or Changed

The generic approved native-tooling closure requirement remains mandatory.
Toolkit runtime contracts and historical evidence retain their meaning. Default
gate mode now performs complete evidence reconciliation and rejects unresolved
coverage on clean canonical dev. Slice 7 mode validates the evidence machinery
while preserving the three blocked rows. A passing
evidence assertion for an expected failure cannot establish a successful journey.

## Files of Interest

- [Project entrypoint](../../../../../README.md)
- [Documentation index](../../../../../docs/README.md)
- [Package map](../../../../../docs/reference/package-map/README.md)
- [Coverage record](../../documentation-conformance-coverage.md)
- [Documentation gate](../../../../../scripts/check-documentation-conformance.sh)
- [Acceptance mapping](../../documentation-conformance-acceptance.md)
- [Integrated receipt binding](../../../../../scripts/documentation-conformance/acceptance.go)
- [Native executable context](../../../../../scripts/documentation-conformance/native-tools.sh)

## Validation

Focused T7.1 repository documentation, licensing, strict lint and independent
451-identity/navigation reconciliation passed during implementation. Required
Slice 7 and exact clean-head whitespace evidence will be recorded through the
controller after task commits and canonical PR-reference follow-up.

T7.2 control tests reject unresolved closure, unbound examples/pages, inherited
lookup directories, automatic toolchain selection, missing executables and
changed images. Focused native execution checks passed the Go link extractor
and all four generator tests, including the exact root command workflow; owned
PostgreSQL stopped. Required documentation, licensing, strict lint and uncached
checker controls passed. Their exact clean-head cumulative results remain
pending until final controller validation. The default integrated gate belongs
to the controller after canonical Slice 7 integration:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 7`
- `git diff --check`

## Risks and Follow-ups

Tickets TKT-20261008134713, TKT-20261008135600 and TKT-20261008135846 block
the affected generator journey. DC07/DC08 claims and integrated DC09/DC10
closure remain unresolved. Fixture interpretation proves local production-kernel
behavior and establishes no authenticated model acceptance.

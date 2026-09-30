<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 1: Builder Contracts

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `docs/hatmax-builder-contracts`
PR: #56

## Purpose

Define one approved, non-conflicting contract set for canonical application
creation and the conversational Hatmax product before runtime implementation.

## Delivered Behavior

Hatmax now has an approved `create_application` contract for creating a named
application from its parent directory. The contract separates display name,
project slug, module path, and package identity; admits targets without silent
overwrite; places composition in `internal/application`; keeps `main.go` thin;
and delays migrator registration until the first real migration exists.

The approved conversational surface defines `hm` as the future canonical TUI
and headless executable while preserving the delivered `hatmax` command during
a bounded transition. It defines open conversation with Book-closed mutation,
adaptive clarification, mandatory plans and approval, resident Codex reuse,
headless parity, cancellation, and recovery.

The companion model defines versioned user-local JSON state, exact
platform-specific roots, scope isolation, atomic snapshots, per-scope locking,
bounded retention and replay, explicit reset and selection, and no durable
approval.

## Implementation Notes

The contracts distinguish approved future behavior from delivered runtime
behavior. Book release 1 and intent schema version 2 remain authoritative for
the current CLI. Application creation enters Book release 2 with intent schema
version 3. The conversational contract becomes runtime authority only after
its implementation slices are delivered.

Bubble Tea, Bubbles, and Lip Gloss v2 are selected only for terminal
presentation. They do not own semantic state and do not enter headless or
generator kernels. No dependency or runtime code was changed in this slice.

## Contracts Added or Changed

- `server_rendered_hatmax_application` owns the canonical application
  scaffold and `internal/application` composition root.
- Application creation does not initialize Git, invent a bootstrap migration,
  infer a hosting owner, domain feature, README, or Diataxis tree.
- Pre-project inventory, fingerprints, staging, publication, and drift use the
  existing planning and execution pipeline.
- `hm conversation new`, `list`, and `resume` control local conversation state
  without modifying project source.
- Conversation storage retains at most 200 turns, 8 MiB of turn content, 50
  operation summaries, and ten conversations per scope.
- Current and successor product surfaces have explicit precedence.

## Files of Interest

- `ops/default/spec/canonical-application-scaffold.md`
- `ops/default/spec/conversational-hatmax-surface.md`
- `ops/default/spec/conversational-hatmax-surface-model.md`
- `ops/default/spec/hatmax-book.md`
- `ops/default/spec/intent-and-planning.md`
- `ops/default/spec/execution-and-conformance.md`
- `ops/default/spec/server-rendered-crud.md`
- `ops/default/spec/interactive-product-surface.md`

## Validation

- `make docs-check` passed.
- `git diff --check` passed.
- Active specification references contain no paths to the promoted draft
  locations.

## Risks and Follow-ups

- Slice 7 must validate whether compact and expanded plan presentations remain
  clear in the TUI prototype.
- A later approved contract must select any additional Hatmax-native
  maintenance capability.
- Runtime support for Book release 2, intent schema version 3, local state, and
  `hm` belongs to Slices 2 through 7.

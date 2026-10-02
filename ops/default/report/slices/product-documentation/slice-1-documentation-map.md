<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Documentation Map

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/product-documentation`
PR: `#4`

## Purpose

Open the Hatmax documentation entry, the User Guide entry, and the reference
vocabulary without writing a chapter or a package contract.

## Delivered Behavior

A reader can open `docs/index.md` and reach the User Guide and the reference.
The User Guide index names the eight chapters and does not link to chapter
files that do not exist yet. Its appendices link to Terminology and the
reference index. The root README links the same two entries and still links
the feature list and the gallery.

## Implementation Notes

Chapter titles are listed as plain text. Appendices link only to pages this
slice added. The feature list, gallery, and changelog stay at their current
paths.

## Contracts Added or Changed

The terminology page records component, transactional startup, static
configuration, settings, and Postgres-backed services. Package readmes stay
the implementation notes.

## Files of Interest

- `docs/index.md`
- `docs/tutorials/index.md`
- `docs/tutorials/user-guide/index.md`
- `docs/reference/index.md`
- `docs/reference/terminology/index.md`
- `README.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in the new Markdown resolve to files in the tree.

## Risks and Follow-ups

None recorded.

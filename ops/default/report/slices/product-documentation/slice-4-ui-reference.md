<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 4: UI Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-4-ui-reference`
PR: `#7`

## Purpose

Publish the UI kit, modal, format, pagination, and internationalization
contracts, and list them from the reference index.

## Delivered Behavior

A reader can open the five subjects from the reference index. The UI page
states the kit, assets, variants, component constructors, and template
functions. The other pages state dialog data, English number and price
formatting, page bounds, and YAML translation lookup.

## Implementation Notes

`modal` stores dialog data and does not render HTML. `pagination` does not
read an HTTP request. `NewResult` divides by `PageSize`, so a zero page size
panics.

## Contracts Added or Changed

Reference pages for `ui`, `modal`, `format`, `pagination`, and `i18n`. No Go
contracts changed.

## Files of Interest

- `docs/reference/ui/index.md`
- `docs/reference/modal/index.md`
- `docs/reference/format/index.md`
- `docs/reference/pagination/index.md`
- `docs/reference/i18n/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

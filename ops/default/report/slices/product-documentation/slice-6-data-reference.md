<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 6: Data Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-6-data-reference`
PR: `#9`

## Purpose

Publish the database, model, validation, seed, and slug contracts, and list
them from the reference index.

## Delivered Behavior

A reader can open the five subjects from the reference index. The pages state
the Postgres connection and migration runner, identifiers and bcrypt helpers,
field validation, one-time seeders, and slug normalization.

## Implementation Notes

`model.HasRole` treats `superadmin` as a bypass. `auth.User.HasRole` does
not. Migration `Down` sections are parsed and are not executed. A seed that
succeeds and then fails to record its name can run again.

## Contracts Added or Changed

Reference pages for `db`, `model`, `validation`, `seed`, and `slug`. No Go
contracts changed.

## Files of Interest

- `docs/reference/database/index.md`
- `docs/reference/model/index.md`
- `docs/reference/validation/index.md`
- `docs/reference/seed/index.md`
- `docs/reference/slug/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

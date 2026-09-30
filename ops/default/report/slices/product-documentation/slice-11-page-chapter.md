<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 11: Page Chapter

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-11-page-chapter`
PR: `#14`

## Purpose

Teach one HTML page and one HTMX update, and list that chapter from the
User Guide.

## Delivered Behavior

A reader can open Serve a Page from the User Guide index. The page contains
a button with `hx-get="/status"`. `GET /status` returns `<p>Updated</p>`.
Choosing Update replaces `#status` with that fragment.

## Implementation Notes

The template manager starts after the database and before route registration.
The page component only implements `RegisterRoutes`. The chapter does not
install a template function.

## Contracts Added or Changed

The Serve a Page chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/pages.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

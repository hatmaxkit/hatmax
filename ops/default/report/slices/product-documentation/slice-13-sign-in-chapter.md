<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 13: Sign-in Chapter

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-13-sign-in-chapter`
PR: `#16`

## Purpose

Teach a page that only a signed-in request can see, and list that chapter
from the User Guide.

## Delivered Behavior

A reader can open Sign In from the User Guide index. `GET /private` without a
session cookie redirects to `/signin` with status `303`. After signup and
sign-in, the same request with the stored cookie receives the private page
containing the account email.

## Implementation Notes

The sample implements `auth.Queries` with two Postgres tables. Password hashes
and session tokens come from `model`. The chapter links to the crypto
reference and does not call those functions. `SetSessionCookie` sets `Secure`,
so the check uses curl rather than a browser on `http://localhost`.

## Contracts Added or Changed

The Sign In chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/sign-in.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

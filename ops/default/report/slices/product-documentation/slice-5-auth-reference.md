<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Auth Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-5-auth-reference`
PR: `#8`

## Purpose

Publish the authentication and crypto contracts, and list them from the
reference index.

## Delivered Behavior

A reader can open both subjects from the reference index. The authentication
page states users, sessions, the queries boundary, service errors, cookies,
and the auth middleware. The crypto page states authenticated strings, lookup
hashes, Argon2id, PASETO claims, and TOTP material.

## Implementation Notes

Signup and sign-in hash passwords through `model`, not through `crypto`.
Session tokens are model IDs. `EncryptEmail`, `DecryptEmail`, and
`ComputeLookupHash` remain in the reference as compatibility functions.

## Contracts Added or Changed

Reference pages for `auth` and `crypto`. No Go contracts changed.

## Files of Interest

- `docs/reference/authentication/index.md`
- `docs/reference/crypto/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

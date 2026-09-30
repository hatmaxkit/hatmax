<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 3: Web Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-3-web-reference`
PR: `#6`

## Purpose

Publish the HTTP, HTMX, middleware, and rendering contracts, and list them
from the reference index.

## Delivered Behavior

A reader can open the four subjects from the reference index. The pages state
form parsing, form errors, template loading, HTMX attributes and headers,
middleware decisions, and template functions.

## Implementation Notes

`middleware/readme.md` shows `RateLimit(100, time.Minute)`. The function
accepts a `*RateLimiter`. The reference follows that signature.

`TriggerEvent` writes the event name. JSON is used by `TriggerEventWithData`
and `TriggerEvents`.

## Contracts Added or Changed

Reference pages for `web`, `htmx`, `middleware`, and `render`. No Go
contracts changed.

## Files of Interest

- `docs/reference/http/index.md`
- `docs/reference/htmx/index.md`
- `docs/reference/middleware/index.md`
- `docs/reference/rendering/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

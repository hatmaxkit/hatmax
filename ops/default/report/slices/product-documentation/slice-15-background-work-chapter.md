<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 15: Background-work Chapter

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-15-background-work-chapter`
PR: `#18`

## Purpose

Teach one background effect caused by a saved note, and list that chapter
from the User Guide.

## Delivered Behavior

A reader can open Work Outside the Request from the User Guide index. Saving
`Hello` returns before the subscriber runs. After the poll interval,
`GET /effects` prints `Hello`.

## Implementation Notes

The sample uses the Postgres pubsub broker. The scheduler and the mailer are
linked and are not exercised. The received titles are kept in memory.

## Contracts Added or Changed

The Work Outside the Request chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/background-work.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

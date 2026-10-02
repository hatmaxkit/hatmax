<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 8: Infrastructure Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-8-infrastructure-reference`
PR: `#11`

## Purpose

Publish the mailer, pubsub, scheduler, telemetry, test helper, and fake
contracts, and close the reference index.

## Delivered Behavior

A reader can open the six subjects from the reference index. Every
first-party package named by this delivery plan has a reference page.

## Implementation Notes

`scheduler` has no package readme, so its page does not link one. The runner
does not call `UpdateNextRun` and does not apply `RetryAttempts`.
`NewMemoryBroker` ignores a handler error. Mailer modes `disabled` and
`dry_run` resolve to `NoopMailer`.

## Contracts Added or Changed

Reference pages for `mailer`, `pubsub`, `scheduler`, `telemetry`,
`testhelper`, and `fake`. No Go contracts changed.

## Files of Interest

- `docs/reference/mailer/index.md`
- `docs/reference/pubsub/index.md`
- `docs/reference/scheduler/index.md`
- `docs/reference/telemetry/index.md`
- `docs/reference/testhelper/index.md`
- `docs/reference/fake/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

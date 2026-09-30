<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 2: Core Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-2-core-reference`
PR: `#5`

## Purpose

Publish the application lifecycle, configuration, and logging contracts, and
list them from the reference index.

## Delivered Behavior

A reader can open the three subjects from the reference index. The lifecycle
page states how setup, start, rollback, serve, and shutdown use the component
lists. The configuration page separates static `config.Config` from runtime
settings. The logging page states the logger methods, levels, and output.

## Implementation Notes

Rollback is documented as the stop-list index walk that `app.Start` executes.
The package comment describes stopping the components that already started.
Those descriptions match only when the start list and the stop list have the
same component at each index.

`Service.GetString` returns the schema default and a nil error when the store
returns an error. `Service.Set` validates a value only when the key is
registered.

## Contracts Added or Changed

Reference pages for `app`, `config`, `settings`, and `log`. No Go contracts
changed.

## Files of Interest

- `docs/reference/application-lifecycle/index.md`
- `docs/reference/configuration/index.md`
- `docs/reference/logging/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

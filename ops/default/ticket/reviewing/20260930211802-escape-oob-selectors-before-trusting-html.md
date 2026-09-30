---
id: TKT-20260930211802
title: Escape OOB attributes before trusting HTML
status: reviewing
kind: bug
severity: high
priority: high
scope: ui
tags: architecture-review, ui, hardening
source: review
reported_at: 2026-09-30T21:18:02Z
ready_at: 2026-09-30T23:04:07Z
started_at: 2026-09-30T23:04:07Z
reviewed_at: 2026-09-30T23:11:12Z
branch: fix/ticket-20260930211802-oob-escaping
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

OOB.Attr and OOBWrapper.Open interpolate selectors into quoted attributes and return trusted template types. A selector containing a quote creates an additional HTML attribute.

Sources: `htmx/oob.go:81-82, htmx/oob.go:124-139`.

Evidence: TestObservedOOBInjection reproduced a new data-review-injected attribute through both public rendering methods.

Impact: Untrusted selector data can escape its attribute context and introduce executable markup. Returning template.HTMLAttr or template.HTML bypasses the template engine's escaping.

## Expected Outcome

Escape attribute values at the final rendering boundary and constrain wrapper tag names. Keep ordinary HTMX selector syntax intact.

## Validation

Cover quotes, ampersands, angle brackets, event attributes, and invalid wrapper tags. Verify normal selectors and IDs still render correctly.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f3).

## Implementation

`Attr` and `Open` escape the complete swap value at the HTML rendering boundary.
`String` remains unescaped, preserving CSS selector syntax. Wrapper tags use an
explicit paired-content allowlist, normalize to lowercase, and default to `div`
when empty. Unsupported tags panic before mutation; both rendering methods
also validate the tag before returning trusted HTML.

Delivery: [OOB rendering safety](../../report/20260930231112-oob-rendering-safety.md).

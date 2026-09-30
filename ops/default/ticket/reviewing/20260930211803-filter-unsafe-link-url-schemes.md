---
id: TKT-20260930211803
title: Filter unsafe URL schemes in trusted link rendering
status: reviewing
kind: bug
severity: high
priority: high
scope: ui
tags: architecture-review, ui, hardening
source: review
reported_at: 2026-09-30T21:18:03Z
ready_at: 2026-09-30T23:38:26Z
started_at: 2026-09-30T23:38:26Z
reviewed_at: 2026-09-30T23:45:18Z
branch: fix/ticket-20260930211803-safe-ui-urls
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Link.Render HTML-escapes href but does not apply URL-context filtering before returning template.HTML. javascript:alert(1) is rendered unchanged; a normal html/template href renders the same value as #ZgotmplZ.

Sources: `ui/link.go:117-154`.

Evidence: TestObservedLinkScheme compares the public link renderer with html/template's URL filtering.

Impact: User-provided URLs can execute script when a rendered link is followed. HTML escaping does not make an executable URL safe.

## Expected Outcome

Apply a defined safe URL policy before constructing trusted HTML. Review equivalent URL-bearing UI primitives for the same boundary.

## Validation

Cover relative URLs, anchors, HTTPS, deliberately supported schemes, mixed-case or obfuscated unsafe schemes, and template integration.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f4).

## Implementation

Native URL attributes in links, navigation, breadcrumbs, and form actions now
use a shared private rendering boundary backed by `html/template`. Plain strings
receive the standard scheme filter, URL normalization, and HTML escaping.
Unsupported schemes become `#ZgotmplZ`; existing empty-value behavior remains
unchanged. Custom trusted HTML and HTMX attributes are outside this policy.

Delivery: [UI URL rendering safety](../../report/20260930234518-ui-url-rendering-safety.md).

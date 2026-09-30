<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# UI URL Rendering Safety

Status: reviewing
Ticket: [TKT-20260930211803](../ticket/reviewing/20260930211803-filter-unsafe-link-url-schemes.md)
Branch: `fix/ticket-20260930211803-safe-ui-urls`
PR: pending

## Delivered Behavior

- `Link`, `Nav`, `NavGrid`, and `PageHeader` breadcrumbs filter native `href`
  values before returning trusted HTML.
- `Form`, `DeleteButton`, and `SettingsForm` apply the same boundary to native
  `action` values. All seven native URL render sites were inspected and updated.
- Relative URLs and case-insensitive `http`, `https`, and `mailto` schemes
  follow Go's contextual URL policy. Unsupported schemes become `#ZgotmplZ`.
- Allowed values receive standard URL normalization and HTML escaping, including
  percent-encoding of quotes, controls, spaces, and non-ASCII bytes. Existing
  valid percent escapes are preserved; entity-looking data is not decoded twice.
- Empty link URLs, omitted form actions, and breadcrumb labels without links
  retain their existing behavior. Constructors and public signatures are unchanged.

## Contracts and Ownership

A private, fixed template establishes quoted URL context. It is parsed once
and never modified by application code; each render uses its own in-memory
writer. Inputs are plain strings, so callers cannot bypass filtering with a
trusted `template.URL` value. The shared boundary delegates to the standard
library rather than maintaining a second URL parser or scheme filter.

The policy follows [Go's contextual URL filtering](https://go.dev/src/html/template/url.go).
It is deliberately conservative about a colon before the first slash, even in
fragment or query text. This restriction, supported schemes, normalization,
and empty-value behavior are documented in the
[UI reference](../../../docs/reference/ui/README.md#url-attributes) and `Unreleased`.
No dependencies were added.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`.

- Before the fix, `go test ./ui -run '^TestURLPolicy/javascript$' -count=1`
  failed for all seven primitives: each emitted `javascript:alert(1)` rather
  than the direct template's `#ZgotmplZ` result.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.1%; `ui`: 97.3%. Database tests
  used an isolated native PostgreSQL cluster on loopback, stopped after validation.
- `make docs-check`: passed.
- `go test ./ui -count=1`: passed.
- `go test -race ./ui ./htmx ./render -count=1`: passed.
- `go test ./ui -run '^$' -fuzz '^FuzzUIURLs$' -fuzztime=10s -parallel=2`:
  passed, 25,668 executions.
- `git diff --check`: passed.

Named cases cover relative URLs, anchors, queries, HTTP(S), mailto, unsupported
schemes, case changes, embedded/leading controls, entity-looking input, encoded
colons, quotes, Unicode, and empty values. A direct `html/template` supplies the
URL-context oracle. Every component is rendered through an outer template and
its URL attributes are decoded with the standard library. Fuzzing also covers
malformed bytes. These checks do not claim browser integration coverage.

## Boundary

Only finding F4 is addressed. External HTTP destinations remain permitted;
this is not an origin allowlist, URL validity check, or application authorization
policy. Arbitrary caller-supplied trusted HTML, custom HTMX attributes, and other
component attributes remain application-owned. The other architecture-review
findings are unchanged.

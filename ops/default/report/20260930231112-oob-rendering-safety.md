<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# OOB Rendering Safety

Status: reviewing
Ticket: [TKT-20260930211802](../ticket/reviewing/20260930211802-escape-oob-selectors-before-trusting-html.md)
Branch: `fix/ticket-20260930211802-oob-escaping`
PR: [#70](https://forge.adrianpk.com/hatmax/hatmax/pulls/70)
Implementation: `3b466c5f4c3611e3552d0cc60336f6f4aafc9d48`

## Delivered Behavior

- `OOB.Attr` and `OOBWrapper.Open` HTML-escape the complete swap value once
  before returning trusted template types. Quotes cannot add attributes or
  elements; literal entity-looking data remains literal after decoding.
- `OOB.String` keeps its raw value. Valid quoted CSS attribute selectors,
  ampersands, combinators, IDs, and swap strategies retain their meaning.
- Wrapper tags use an explicit paired HTML content allowlist, including table,
  list, and template elements. Names normalize to lowercase; an empty name
  selects `div`.
- Malformed or unsupported tags panic before changing the builder. Rendering
  validates the tag again, excluding raw-text, embedded, void, custom, and
  foreign/namespaced elements from trusted fragments.
- Existing ID and class escaping is preserved and covered by regression tests.

## Contracts and Ownership

Public signatures and ordinary wrapper output are unchanged. Unsupported tags
now panic instead of becoming trusted markup; this compatibility restriction
is documented in the [HTMX reference](../../../docs/reference/htmx/README.md#attributes)
and `Unreleased`. Callers own tag selection and valid HTML nesting.

The fix follows the trust boundary documented by
[Go's template package](https://pkg.go.dev/html/template#HTMLAttr): trusted HTML
types bypass contextual escaping, so builders must escape their data first.
Table and list wrappers remain supported for
[HTMX out-of-band swaps](https://htmx.org/attributes/hx-swap-oob/).
No dependencies were added.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`.

- Before the fix, `go test ./htmx -run '^TestOOBEscape$' -count=1` failed:
  both public methods emitted injected `onfocus` and `data-review-injected`
  attributes, and quoted CSS selectors produced malformed markup.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.1%. Database tests used an
  isolated native PostgreSQL cluster on loopback, stopped after validation.
- `make docs-check`: passed.
- `go test ./htmx ./render -count=1`: passed.
- `go test -race ./htmx ./render -count=1`: passed.
- `go test ./htmx -run '^$' -fuzz '^FuzzOOBEscape$' -fuzztime=10s -parallel=2`:
  passed, 159,355 executions.
- `git diff --check`: passed.

Tests render through `html/template` and strictly decode paired fragments with
the standard library to check their element count, attribute count, and decoded
values. Fuzz input is limited to XML-compatible characters for that decoder;
this is not a browser integration test. Named cases cover every allowed tag,
tag injection, event/element injection, entity preservation, case normalization,
empty defaults, and rejection without builder mutation.

## Boundary

Only finding F3 is addressed. This change does not sanitize content supplied
between wrapper tags, authorize CSS targets, or change other HTMX builders,
link URLs, or the remaining architecture-review findings.

<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Web Interaction

Status: delivered
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-web-interaction`
PR: #49

## Purpose

Explain how requests cross Hatmax HTTP boundaries and become full-page or
targeted HTMX responses, then connect forms, validation, and presentation
helpers as one server-owned interaction model.

## Delivered Behavior

The User Guide now follows an HTTP request from application-wide and grouped
middleware through a thin handler and into an application service. It places
route visibility after successful startup, same-origin protection before
state-changing handlers, and error translation at the request boundary.

Pages and partials share one embedded template set, feature namespace, view
model, and server-rendered representation. The guide explains explicit HTMX
request detection, stable partial replacement units, ordinary HTTP fallbacks,
redirect behavior, and trusted-HTML boundaries.

Forms now progress through parsing, normalization, structured Hatmax
validation, service-owned rules, persistence constraints, and safe feedback.
The presentation chapter then places UI components, modal configuration,
formatting, pagination, and internationalization around application-owned
view models and templates.

## Implementation Notes

The chapters use one invoice vocabulary without instructing the reader to
build a cumulative sample application. Short examples establish the
responsibility and assembly of each primitive; exact builders, rules, and
edge cases remain in reference documentation.

The superseded page and form exercises were removed after their durable HTMX,
same-origin, rendering, validation, and recovery material was incorporated.
Inbound links now enter the technical journey instead of the retired exercise
sequence.

## Contracts Added or Changed

- Middleware establishes request-wide facts and grouped route policy before a
  handler translates HTTP input.
- Full pages and HTMX partials share server-owned state, templates, and
  application behavior.
- Browser validation is an interaction aid; Hatmax server validation remains
  authoritative.
- Validation rules live at the boundary whose knowledge they require.
- Presentation primitives support application-owned view models rather than
  introducing a separate frontend architecture.
- Trusted HTML and attributes are limited to Hatmax or application-controlled
  markup.

## Files of Interest

- `docs/tutorials/user-guide/request-boundary.md`
- `docs/tutorials/user-guide/pages-and-partials.md`
- `docs/tutorials/user-guide/forms-and-validation.md`
- `docs/tutorials/user-guide/presentation-primitives.md`
- `docs/tutorials/user-guide/README.md`

## Validation

- `make docs-check` passed.
- Request and presentation coverage audit passed for `middleware`, `web`,
  `render`, `htmx`, `validation`, `ui`, `modal`, `format`, `pagination`, and
  `i18n`.
- Superseded `pages.md` and `forms.md` link audit passed.
- `git diff --check` passed.
- Pull request #49 merged into `dev` at
  `cf61fe0a024e68c73b91a4ea6c17a24c626f14ec`.

## Risks and Follow-ups

The interaction chapters deliberately stop before assembling a complete
feature. Slice 3 connects these request, response, and validation boundaries
to the canonical model, store, service, handler, template, wiring, and test
flow.

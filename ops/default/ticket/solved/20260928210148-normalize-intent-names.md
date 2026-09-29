---
id: TKT-20260928210148
title: Normalize canonical names outside the interpreter model
status: solved
kind: bug
severity: high
priority: normal
scope: domain
tags: generator, intent, determinism
source: manual_test
reported_at: 2026-09-28T21:01:48Z
ready_at: 2026-09-29T07:02:59Z
started_at: 2026-09-29T07:24:33Z
reviewed_at: 2026-09-29T07:29:31Z
closed_at: 2026-09-29T07:29:57Z
resolution: fixed
branch: dev
commits: c36021d528137ce6580197f4ac74ebc54d825007
---

## Observed Behavior

Natural requests to create an invoice feature were rejected because the
interpreter returned `invoice` where the intent validator required the
exported Go identifier `Invoice`. Field-label fallback also preserves a raw
snake-case or lowercase name instead of producing the canonical display
label. A user currently has to phrase implementation-level identifiers to
obtain an admitted intent.

## Expected Outcome

The interpreter should identify semantic concepts while Hatmax
deterministically derives or normalizes canonical feature names, exported Go
identifiers, routes, plurals, and default display labels. Equivalent natural
requests must admit the same normalized intent and plan without requiring Go
syntax in the prompt.

## Validation

- Add paraphrase cases for ordinary create-feature requests.
- Prove equivalent requests normalize to the same intent and plan digest.
- Prove explicit valid product labels remain distinguishable from defaults.
- Keep invalid or materially ambiguous names rejected with stable diagnostics.

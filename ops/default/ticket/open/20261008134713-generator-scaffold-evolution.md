---
id: TKT-20261008134713
title: Recognize generated application composition roots for feature evolution
status: open
kind: bug
severity: unclassified
priority: unclassified
scope: api
tags: generator, composition-root
source: implementation
reported_at: 2026-10-08T13:47:13Z
commits: 2d8f46019c7fed059c5909be6ff5f14a875c403e
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

Slice 6 native workflow execution created both the published bare Ledger
scaffold and a Ledger scaffold with an invoice feature. Creation, compilation,
checks and conversation rebinding succeeded. Approving a subsequent feature
creation or required timestamp returned `HMGEN-EXECUTION-LAYOUT-MISSING`:
`project has no Hatmax composition root`. No follow-up source mutation occurred.

`generator/project/layout.go` records composition roots only in `package main`
with a `main` function and Hatmax `Setup` call. The scaffold's thin main delegates
to `internal/application`, where its actual composition lives. Feature layout
resolution cannot select it. The full create-then-evolve documentation claim is
blocked; documentation work does not authorize this runtime correction.

## Expected Outcome

Recognize the canonical generated application composition without moving wiring
into main or weakening thin-main conformance. Create and evolve a generated
application through the same conversation and headless kernel with real tools.

## Validation

Exercise bare and initial-feature scaffolds with real Go, SQLC and project
checks. Resume the created conversation, approve a feature or required field,
and verify generated SQL/model/handler/wiring and passing project checks.
Retain rejection of ambiguous composition roots and stale-plan mutation.

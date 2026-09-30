---
id: TKT-20260930211814
title: Bound image ingestion before buffering and decoding
status: open
kind: bug
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:14Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Resize reads the complete input without a byte limit and decodes the full image before inspecting dimensions. S3 Put also buffers the entire input and creates an additional string copy. Resize does not consult ctx.

Sources: `image/stdprocessor/processor.go:49-64, image/s3/store.go:77-93`.

Evidence: Source-confirmed; no unbounded allocation or denial-of-service probe was run.

Impact: Applications processing untrusted uploads have no primitive-owned limit on encoded bytes or decoded pixel allocation. Output resize limits do not bound input memory.

## Expected Outcome

Define encoded-size and decoded-pixel limits, inspect image configuration before full decode, and support cancellation where the underlying operation permits it. Avoid unnecessary S3 buffer copies.

## Validation

Use bounded tests for over-limit readers, oversized image dimensions, canceled work, ordinary resizing, and S3 upload behavior.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f15).

---
id: TKT-20260930211814
title: Bound image ingestion before buffering and decoding
status: solved
resolution: fixed
kind: bug
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:14Z
ready_at: 2026-10-01T11:47:23Z
started_at: 2026-10-01T11:47:23Z
reviewed_at: 2026-10-01T12:00:35Z
closed_at: 2026-10-01T12:36:01Z
branch: fix/ticket-20260930211814-image-ingestion
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/82
commits: 45f6c8047d257798df7195db1b45889888eae8b9, 9768c02c9a34cfe269128a865bb143a495c6a6fb
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

## Implementation

Standard resizing and S3 uploads accept at most 20 MiB of encoded input, with one additional byte to detect overflow without draining oversized readers. A shared image-internal reader checks cancellation before and after each underlying read and returns no usable partial buffer on failure. S3 uploads use bytes.NewReader directly rather than copying the buffer into a string.

The standard processor inspects configuration before full decoding and rejects dimensions above 25 million pixels using overflow-safe accounting. GetDimensions preserves header-only inspection with bounded header reads and the same pixel limit. Resize checks cancellation between CPU phases, rejects non-positive target bounds, and clamps target bounds to source dimensions before ratio arithmetic. Constructors and public interfaces remain unchanged.

Readers remain caller-owned. Cancellation cannot preempt an arbitrary blocked Read or codec/scaling computation; applications still own I/O deadlines and processing concurrency. These limitations and the fixed acceptance limits are documented.

Validation passed: make check with Go 1.26.7 and isolated PostgreSQL 18.6 (81.3% total coverage), make docs-check, 20 race-enabled image-package repetitions, and git diff --check. Bounded fixtures cover exact/over byte limits, large PNG dimensions without allocating pixels, long JPEG header scans, cancellation, ordinary resizing, and local S3-compatible uploads.

Delivery: [Bounded image ingestion](../../report/20261001120035-image-ingestion.md).

Merged into `dev` through PR #82 at `9768c02c9a34cfe269128a865bb143a495c6a6fb`.

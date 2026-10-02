<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 7: Media Reference

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-7-media-reference`
PR: `#10`

## Purpose

Publish the image storage and processing contract, and list it from the
reference index.

## Delivered Behavior

A reader can open the image subject from the reference index. The page states
records, variant sizes, the store and repository boundaries, the local and S3
stores, and the standard processor.

## Implementation Notes

`image/readme.md` calls `Resize` with variant fields that the package does
not export. The reference follows `Resize(ctx, input, contentType, maxWidth,
maxHeight)` and `VariantSpec`.

## Contracts Added or Changed

The reference page for `image`, `image/local`, `image/s3`, and
`image/stdprocessor`. No Go contracts changed.

## Files of Interest

- `docs/reference/image/index.md`
- `docs/reference/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

---
id: TKT-20261016193736
title: Prevent concurrent S3 signing and object operation races
status: open
kind: bug
severity: medium
priority: high
scope: api
tags: storage, s3, concurrency
source: trace
reported_at: 2026-10-16T19:37:36Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

At `ba54e7e8147708a7d67b6d4a85fa8450c60a6968`, concurrent calls to
`storage/s3.Store.SignUpload` and `Delete` race inside AWS S3 SDK v1.114.0.
Each signing call constructs a presign client with client options. The SDK
constructor writes the shared Express credentials provider while another
operation clones that provider. The race detector identifies writes in
`finalizeExpressCredentials` and reads in `CloneWithBaseCredentials`.
Concurrent signing calls can also reconstruct the same shared provider.

## Expected Outcome

Initialize signing state before exposing a store to concurrent callers and
reuse it without mutating shared SDK configuration. Keep request-specific
policy binding isolated. Preserve exact destination, type, size and expiry
conditions, renewable credentials, finite operation deadlines and cleanup.
Do not serialize unrelated object operations or introduce another dependency.

## Validation

- Reproduce concurrent signing and signing/deletion with race detection before
  changing production code; verify each concurrent grant's own policy.
- Run affected storage and image regressions, real pinned MinIO private/signed
  upload roots, vet, configured lint, documentation and license checks.
- Preserve public interfaces and existing published module compatibility.

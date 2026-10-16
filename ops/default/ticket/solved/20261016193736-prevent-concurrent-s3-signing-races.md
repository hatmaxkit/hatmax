---
id: TKT-20261016193736
title: Prevent concurrent S3 signing and object operation races
status: solved
kind: bug
severity: medium
priority: high
scope: api
tags: storage, s3, concurrency
source: trace
reported_at: 2026-10-16T19:37:36Z
ready_at: 2026-10-16T19:38:43Z
started_at: 2026-10-16T19:38:43Z
reviewed_at: 2026-10-16T19:41:07Z
closed_at: 2026-10-16T19:42:06Z
resolution: fixed
validated_commit: 46d6781e98dded955995a459c7b8a0d8c70261d2
commits: d065ab6e6a42, 46d6781e98dded955995a459c7b8a0d8c70261d2
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

`TestConcurrentSigning` reproduced the SDK race before the runtime change.
The constructor now initializes one reusable presign client before exposing the
store; each signing call still binds its own policy signer and deadlines.
Public interfaces and dependencies are unchanged.
The correction is integrated in `dev` at the validated commit above.

- `GOWORK=off go test -race -count=10 -run '^TestConcurrentSigning$' -timeout=120s ./storage/s3` passed both concurrent cases and exact policy assertions.
- `GOWORK=off go test -race -count=1 -timeout=180s ./storage/... ./image/...` passed.
- `GOWORK=off go vet ./storage/... ./image/...` passed.
- `GOWORK=off golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./storage/s3` passed with zero issues.
- `GOWORK=off go test -json -race -tags=integration -count=1 -p=1 -parallel=2 -timeout=360s -run '^Test(PrivateObjects|RemoteBounds|RemoteFailures|SignedUploads|SigningExpiry|SnapshotCompletion|ObjectLifecycle)$' ./storage/s3` passed all seven roots with no skips on the approved native MinIO fixture; object/bucket/server/temporary-root cleanup was confirmed.
- `make source-license-check` and `make docs-check` passed.

These are the focused correction and actual provider checks, not a new complete
repository acceptance gate. The previously accepted object-storage candidate
and receipts retain their original identities.

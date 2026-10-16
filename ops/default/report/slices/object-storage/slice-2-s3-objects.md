<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Private S3 Objects

Status: delivered
Delivery set: object-storage
Plan: [Object storage plan](../../../plan/object-storage.md)
Tracker: [Object storage tracker](../../../tracker/object-storage.md)
Branch: `feat/private-s3-storage`
PR: `#124`
Integrated candidate: `c03a4f042507eb8e255571aa2781fcf7a42f0036`
Acceptance completed: 2026-10-16

## Purpose

Deliver bounded private S3 object I/O and inspection through the existing AWS
SDK, with actual server evidence and explicit ownership of input, spool and
response resources.

## Delivered Behavior

- Put, inspect, open and delete empty or non-empty objects in a private bucket
  without public URLs or image records. Anonymous access remains denied.
- Validate keys and limits before I/O. Completely spool bounded input before one
  write attempt; failed or oversized input preserves existing remote bytes.
- Return metadata from the same opened response and independently bound actual
  reads. An already opened response retains its observed object after overwrite.
- Distinguish confirmed absence, authorization denial, malformed metadata and
  uncertain mutations. Lost write/delete acknowledgments remain indeterminate;
  applications can reconcile through later inspection without automatic retries.

## Implementation Notes

T2.1 is `9de7c8aef2d8a487cf192ad3b5899e64157d2fb5`; T2.2 is
`b85c9496d412f0a759e4640264ffa0689c2fdd0c`.
The adapter uses the existing AWS SDK root `v1.47.1` and S3 `v1.114.0`, without
changing product dependencies. SDK credentials support the default chain and
explicit providers. Production endpoints verify TLS certificates; plaintext is
admitted only through an explicit option with a literal loopback address.

Private replayable spool files use exclusive random names and mode 0600 beneath
an owned directory handle. Each write uses a fixed 32 KiB copy buffer and at most
its admitted spool capacity; N concurrent writes can consume N times that disk
capacity plus filesystem overhead. Callers own aggregate quotas and admission.
Supplied remote content type is stored metadata; omitted type is derived from
at most 512 bytes. Neither content metadata nor ETags establish content safety.

The owned Linux fixture was built before integration tests from source
`9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`, module
`github.com/minio/minio v0.0.0-20251015172955-9e49d5e7a648`, checksum
`h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM=`, using Go 1.27.1.
Observed binary SHA-256:
`726b305d2753cc6b5585028b006bde0e6e7f58d7774b1308bc8b4eaa49eb5d6c`.
Preparation retains source/module metadata, full Go build information and the
digest independently of the product module. The source-built executable's
banner is DEVELOPMENT.GOGET; its pinned module/build identity is authoritative.

Fixture startup, readiness and bucket creation share a 60-second deadline.
Transport effects use finite timeouts, bucket cleanup has a 120-second deadline
and server shutdown has a 30-second bound. Ephemeral credentials remain in the
child environment. Receipts retain object counts and log hashes without secrets
or form fields. A Linux parent-death guard also removes the owned server if the
test harness exits abnormally. Only the fixture's own process is signaled.

## Contracts Added or Changed

`storage/s3.NewStore` takes typed `Options` for the private bucket, region,
endpoint, credentials, existing spool directory and capacity. ReadAttempts
defaults to 1 and admits 1-5; OperationTimeout defaults to 30 seconds and admits
at most 120 seconds, including returned readers. Mutations always use one SDK
attempt. Redirects cannot silently replay them. `Close` releases the owned spool
handle and idle HTTP connections after callers finish their readers.

`Put` never closes caller input or starts detached I/O to interrupt a blocked
reader. Cancellation and deadlines retain their causes through error wrapping.
Rejected response bodies are closed before returning; successful response
readers remain caller-owned. Key deletion in a versioned bucket does not erase
historical versions. Neutral storage contracts and historical image APIs retain
their existing forms.

## Files of Interest

- `storage/s3/options.go`: private configuration, TLS and resource ownership.
- `storage/s3/read.go`, `write.go`, `errors.go`: bounded operations and errors.
- `storage/s3/*_test.go`: real private lifecycles and complementary fault cases.
- `scripts/prepare-object-storage-tools.sh`: immutable fixture build and reuse.
- `scripts/check-object-storage.sh`: delivered mode 2 and required-root receipts.

## Validation

Passed during task implementation:

- `scripts/prepare-object-storage-tools.sh`
- `GOWORK=off go test -count=1 -timeout=60s ./storage/s3`
- Actual-server TestPrivateObjects, TestRemoteBounds and TestRemoteFailures,
  including private access, metadata, old-content preservation, snapshot identity,
  and lost confirmations after actual committed writes/deletes.

The focused actual-server command passed with explicit owned fixture inputs:

```sh
HATMAX_STORAGE_MINIO="$PWD/.tmp/object-storage/tools/minio" HATMAX_STORAGE_FIXTURE_RECEIPT="$PWD/.tmp/object-storage/development-fixture.txt" GOWORK=off go test -tags=integration -count=1 -timeout=360s -run '^(TestPrivateObjects|TestRemoteBounds|TestRemoteFailures)$' ./storage/s3
```

The observed fixture receipt confirmed two remaining test objects before
cleanup, zero after cleanup, bucket removal, server shutdown and temporary-root
removal. All three final controller-recorded commands passed at the merged
PR HEAD `8322aa5e654f51ac240c868426be1e565509fb88`:

- `GOWORK=off scripts/check-object-storage.sh slice 2`
- `GOWORK=off make docs-check`
- `git diff --check`

The controller's exact clean-HEAD receipt owns their final result. Mode 2 runs
ordinary/race storage checks, existing image/local and image/s3 regressions,
required tagged S3 roots with race detection, licensing, formatting, shell
syntax, affected build/vet and configured ordinary/integration lint. Missing
roots, skips, failed cleanup or fixture identity drift fail acceptance.

Integrated closure: the controller's
`GOWORK=off scripts/check-object-storage.sh` passed at
`c03a4f042507eb8e255571aa2781fcf7a42f0036` in validation
`ORAZRFW4UDUF4TSQEPDZ47HPFF`. All OS01-OS10 criteria and owned cleanup are
accepted. The canonical [acceptance report](../../object-storage-acceptance.md)
retains the complete command/result manifests and evidence hashes.

## Risks and Follow-ups

The actual-server evidence establishes the pinned MinIO profile, not arbitrary
S3-compatible vendors. The native ownership fixture currently requires Linux;
unsupported hosts fail visibly. Remote mutation failure cannot guarantee
rollback. Callers choose retry and reconciliation policy. Signing, executable
guides and integrated acceptance are delivered through Slices 3-5.

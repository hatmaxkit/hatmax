<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Integrated Acceptance

Status: delivered
Delivery set: object-storage
Plan: [Object storage plan](../../../plan/object-storage.md)
Tracker: [Object storage tracker](../../../tracker/object-storage.md)
Branch: `test/object-storage-acceptance`
PR: `#127`
Integrated candidate: `c03a4f042507eb8e255571aa2781fcf7a42f0036`
Acceptance completed: 2026-10-16

## Purpose

Compose all delivered storage boundaries into OS01-OS10 evidence and enable
exact-candidate integrated acceptance through the existing full repository gate.

## Delivered Behavior

- Execute one caller lifecycle through local and actual S3 adapters: observed
  metadata, opened snapshot identity, distinct final bytes, failed replacement
  preservation, cancellation and idempotent confirmed absence.
- Bind the versioned acceptance ledger to discovered package/tag/root manifests,
  ordered commands, results, source/tool/SDK identities, log hashes and cleanup.
- Reject changed candidates or logs, missing/reordered commands, missing roots,
  required skips and incomplete actual fixture teardown.
- Enable full mode only on clean pushed dev, including the existing make check
  target and a fresh owned native PostgreSQL cluster through existing DB inputs.

## Implementation Notes

T5.1 is `9cde0688161bab2a69ef6b9a49cb62054bef5ae6`; T5.2 is
`bd1bd280a1ae2c7409d748ed0c6953fc795ea1fa`. The canonical
[acceptance report](../../object-storage-acceptance.md) retains the immutable
ledger introduction. Source discovery reads actual build-tag-selected Go files
without running TestMain, and inventories executable tests, examples and fuzz
seeds. Required ordinary/race, remote, guide and sanitized guide-child results
must all pass. Each execution produces 21 ordered command records plus an exact
manifest and hashed acceptance receipt in ignored owned storage.

Source identity includes the full HEAD/tree, all tracked file bytes, branch,
canonical live pushed head, Go 1.27.1, existing AWS SDK/S3 module versions, ledger
and evidence executable digests. Metadata lookup commands are bounded. Step logs
are hashed and parsing bounds each JSON event to 8 MiB; manifests admit at most
16 MiB and 128 command records. Full OS10 acceptance also requires the complete
repository roots and original license/format/vet/test/80%-coverage/strict-lint gate.

The actual MinIO source is `9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`, module
`v0.0.0-20251015172955-9e49d5e7a648`, built with Go 1.27.1; its observed binary
SHA-256 is `726b305d2753cc6b5585028b006bde0e6e7f58d7774b1308bc8b4eaa49eb5d6c`.
Each run verifies its actual identity and both remote/guide cleanup receipts.
The native PostgreSQL fixture observed version 18.6 and records all tool digests,
readiness, log hashes, shutdown and temporary-root removal. TCP is disabled in
a private mode-0700 Unix socket root. Initialization/start/shutdown are bounded
to 60/30/30 seconds, its command to 30 minutes, tagged roots to 360 seconds and
the guide to 600 seconds. Only owned services are stopped.

## Contracts Added or Changed

No product API changes. Makefile TEST_FLAGS optionally changes test output,
freshness and finite scheduling while preserving every existing aggregate-gate
prerequisite and the complete package scope. The runner confines its lint cache
to ignored owned output and fails if formatting or another command changes the
candidate. Explicit unrelated live-provider/subprocess/fuzz-boundary skips are
retained in full results and cannot satisfy an OS root.

The source ticket retains its original ID/filename and moves from in_progress
to reviewing with reviewed_at, task identities and matching source links. It
remains unresolved until the controller's integrated gate and restricted closure.
Slice 5 mode keeps OS10 pending even when all focused evidence is green.

## Files of Interest

- `storage/s3/lifecycle_test.go`: shared local/remote lifecycle and snapshot use.
- `scripts/object-storage/ledger.go`: immutable criterion and ordered-step contract.
- `scripts/object-storage/main.go`, `evidence.go`, `evidence_test.go`: source
  discovery, exact identity/result binding and negative controls.
- `scripts/object-storage/postgres-fixture.sh`: bounded private native database.
- `scripts/check-object-storage.sh`, `Makefile`: focused/full composition and
  complete repository test result capture.
- `scripts/documentation-conformance/object_acceptance_test.go`,
  `object_guide_integration_test.go`: criterion reconciliation and safe child results.
- `ops/default/report/object-storage-acceptance.md`: canonical gate evidence owner.

## Validation

Focused ledger/result/fixture/command controls and source reconciliation passed:

- `GOWORK=off go test -count=1 -timeout=120s ./scripts/object-storage`
- `GOWORK=off go test -count=1 -timeout=120s -run '^TestObjectAcceptance$' ./scripts/documentation-conformance`
- `GOWORK=off go run ./scripts/object-storage contract`

The actual-server lifecycle passed with race detection:

```sh
HATMAX_STORAGE_MINIO="$PWD/.tmp/object-storage/tools/minio" HATMAX_STORAGE_FIXTURE_RECEIPT="$PWD/.tmp/object-storage/development-lifecycle-fixture.txt" GOWORK=off go test -json -race -tags=integration -count=1 -p=1 -parallel=2 -timeout=360s -run '^TestObjectLifecycle$' ./storage/s3
```

The actual signed guide with sanitized child results also passed. Both receipts
confirmed empty spools, zero objects, bucket removal, owned server shutdown and
temporary-root removal. A focused native database fixture executed
`go test -json -count=1 -p=1 -timeout=120s -run '^TestStartAndStopIntegration$' ./db`
through postgres-fixture.sh and passed; its readiness, tool/log hashes, shutdown
and root removal were checked. Configured ordinary/integration lint passed.

Final PR acceptance uses these ordered controller-recorded commands:

- `GOWORK=off scripts/check-object-storage.sh slice 5`
- `git diff --check`

Both commands passed at clean pushed HEAD
`ad83e03b79052ef403c7be33cdc7fc6c27e45675` in controller validation
`IZKKCB7PYSOCATN6P5UHWRSGFW`. Canonical PR #127 merged that commit into dev
on 2026-10-16. The receipt binds 21 ordered steps, 81 discovered roots and
confirmed MinIO/guide/PostgreSQL cleanup, with OS10 pending at slice delivery. The controller subsequently executed the
configured full command after integration; that exact passing evidence accepts
OS10 and authorizes this restricted documentation closure.

Integrated closure: the controller's
`GOWORK=off scripts/check-object-storage.sh` passed at
`c03a4f042507eb8e255571aa2781fcf7a42f0036` in validation
`ORAZRFW4UDUF4TSQEPDZ47HPFF`. All OS01-OS10 criteria and owned cleanup are
accepted. The canonical [acceptance report](../../object-storage-acceptance.md)
retains the complete command/result manifests and evidence hashes.

## Risks and Follow-ups

The integrated gate passed with native PostgreSQL 18.6 and the identified
unversioned MinIO profile. This establishes their exercised compatibility;
arbitrary deployment/provider versions require their own evidence.

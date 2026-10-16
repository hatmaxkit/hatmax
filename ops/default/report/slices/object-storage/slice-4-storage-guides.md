<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Object Storage Guides

Status: reviewing
Delivery set: object-storage
Plan: [Object storage plan](../../../plan/object-storage.md)
Tracker: [Object storage tracker](../../../tracker/object-storage.md)
Branch: `docs/object-storage-workflows`
PR: `#126`

## Purpose

Give applications executable private object workflows, exact configuration and
resource contracts, and a clear boundary between bearer transport and completion.

## Delivered Behavior

- Link a bounded local document tutorial, private S3/signing/reconciliation
  procedures, exact API reference and bearer/snapshot ownership explanation.
- Compile and execute the published local lifecycle and actual signed multipart
  staging-to-distinct-final workflow without logging grants or credentials.
- Reject missing/changed published snippets and dynamic example output. Preserve
  historical image constructors, noncanonical accepted image keys and URL strings.
- Extend cumulative runner mode 4 with documentation checks, all image package
  regressions, required guide roots, race checks and confirmed owned cleanup.

## Implementation Notes

T4.1 is `7161cc892c5f0e56ddd5eb11c9c96d647c8d43ee`; T4.2 is
`270d1a89d8d562dd84f8ec008d014c35fb1592d9`. Documentation follows the repository's
four quadrants and README entrypoints. Source markers bind five published Go
blocks to complete executable files or exact parsed functions. Ordinary checks
compile the real examples and verify fixed literal output; negative controls
reject grant fields and provider-body logging.

The integration,documentation guide root delegates to a required actual-server
root that uses the existing pinned native fixture. The example retrieves its
credentials only through the SDK environment chain, submits the returned form,
reads staging to complete bounded EOF, validates actual bytes, stores those same
bytes under a distinct final key, reads them back and removes both exercise keys.
Fixture credentials stay in child environment only; child output is checked
against a fixed message and never copied into failure diagnostics.

The identified source remains `9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`,
module `v0.0.0-20251015172955-9e49d5e7a648`, Go 1.27.1 and observed binary
SHA-256 `726b305d2753cc6b5585028b006bde0e6e7f58d7774b1308bc8b4eaa49eb5d6c`.
Existing exact binaries are reused only after module/build/checksum verification.
No external account or existing service supplies acceptance.

## Contracts Added or Changed

No product API changes. The reference states exact key/MIME/int64/expiry bounds,
constructor-owned endpoint/bucket/credentials, finite per-operation limits,
reader/spool ownership, metadata provenance and classified uncertainty. The
signing profile remains an explicit deployment assertion for the identified
unversioned MinIO release; arbitrary vendors are not implicitly validated.

Examples use a finite 1 MiB snapshot budget and fixed small multipart payload.
They do not mark application records available. Applications own authority,
aggregate quotas, format validation, immutable destinations and durable recovery.
Legacy image/local URL double-slash behavior and its accepted space-containing
key remain intact; image/s3 constructor/URL behavior and all processor regressions
are retained without adapting their public interfaces to storage.Store.

## Files of Interest

- `docs/tutorials/private-objects/README.md`: complete local learning workflow.
- `docs/how-to/store-private-objects/README.md`, `sign-object-uploads/README.md`,
  `reconcile-object-storage/README.md`: task procedures and real POST submission.
- `docs/reference/storage/README.md`,
  `docs/explanation/private-object-ownership/README.md`: exact contracts and owners.
- `examples/object-storage/`: runnable local and remote compositions.
- `scripts/documentation-conformance/object_guide_test.go`,
  `object_guide_integration_test.go`, `storage/s3/guide_test.go`: source/output
  controls, compatibility and actual remote execution with owned cleanup.
- `scripts/check-object-storage.sh`: cumulative mode 4 and finite guide receipts.

## Validation

Focused example compilation and ordinary guide/compatibility checks passed:

- `GOWORK=off go test -run '^$' ./examples/object-storage/...`
- `GOWORK=off go test -count=1 -timeout=120s -run '^TestObject(GuideDrift|GuideControls|LocalGuide|ImageCompatibility)$' ./scripts/documentation-conformance`

The focused actual-server walkthrough passed with race detection:

```sh
HATMAX_STORAGE_MINIO="$PWD/.tmp/object-storage/tools/minio" HATMAX_STORAGE_FIXTURE_RECEIPT="$PWD/.tmp/object-storage/development-guide-fixture.txt" GOWORK=off go test -json -race -tags=integration,documentation -count=1 -p=1 -parallel=2 -timeout=600s -run '^TestObjectStorageGuide$' ./scripts/documentation-conformance
```

Its receipt confirms exact final readback, an empty spool, zero objects before
and after fixture cleanup, bucket removal, owned server shutdown and temporary
root removal. Configuration-aware ordinary/integration lint passed.

Initial cumulative validation stopped after successful ordinary roots because
Go emitted package-level skip events for the two executable examples without
unit-test files. Examples now compile/build/vet/lint separately from required
test-root execution; required-root skip rejection remains unchanged.

Final PR acceptance uses these ordered controller-recorded commands:

- `GOWORK=off scripts/check-object-storage.sh slice 4`
- `git diff --check`

The exact clean-HEAD controller receipt owns their final result. Mode 4 includes
make docs-check, all prior local/remote/signing roots, all image packages,
TestObjectGuideDrift and the tagged TestObjectStorageGuide without required skips.
Guide execution has the approved 600-second bound; earlier tagged roots retain
360 seconds. Full mode remains unavailable until its approved Slice 5 owner.

## Risks and Follow-ups

The example's fixed exercise keys require an isolated owned bucket. Its content
check demonstrates validating known bytes, not a general document-safety policy.
Production admission, aggregate bounds and durable completion remain application
responsibilities. Complete OS01-OS10 acceptance retains its approved Slice 5 owner.

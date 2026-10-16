<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Delivery Plan

Date: 2026-10-16
Status: Active
Approved: 2026-10-16
Specification approved: 2026-10-16
Planning authorized: 2026-10-16
Delivery set: object-storage
Concern: [Object storage capabilities](../spec/object-storage.md)
Source ticket: [Bounded object storage and signed uploads](../ticket/reviewing/20261016081501-support-bounded-object-storage-and-signed-uploads.md)
Tracker: [Delivery tracker](../tracker/object-storage.md)
Base branch: `dev`
Planning base: `60fcca7f8f88059608b353420dfa7513c0b78279`
Execution authorized: 2026-10-16; five-slice Looper delivery, exact-candidate full gate and restricted closure
Integration authorized: Controller-owned Rebase + Fast-forward into dev
Active slice: None; all five slices delivered
Next slice: None; final gate pending
Active tasks: None; awaiting controller-owned full gate
Execution gate: Open
Worker budget: 300 total turns; 10 consecutive stalled turns
Slice strategy: behavior-first
Reason: local I/O, remote I/O and signed forms each deliver complete package behavior and real boundary evidence without empty interfaces or intentionally failing layers.
Go baseline: 1.27.1; GOWORK=off

## Scope and Activation

Deliver OS01-OS10 through small neutral storage contracts, local and S3 adapters,
exact-target signed POST, image compatibility and executable documentation.
Application authorization, ownership, quotas, durable completion and retry policy
remain caller responsibilities. This delivery set is owned entirely by this
repository, with no external application's state or controller as a prerequisite.

The maintainer approved this plan and authorized publication and five-slice
Looper execution on 2026-10-16, including controller-owned Rebase + Fast-forward
merges, the complete repository gate and restricted closure. Verify clean dev,
actual canonical origin/base, open PR/worktree inventory, installed tools and
exclusive controller ownership. Read installed `looper --help` at activation.
The authorization is for this exact delivery set; it does not add release work.

The repository has no scheduled nightly integration. Its `.agents/PLAYBOOKS.md`
owns local validation and later main alignment. This set's delivery targets dev;
main promotion, tags, releases and backup publication are separate operations.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Bounded local objects | `feat/bounded-local-storage` | `feat(slice-1): store bounded private local objects` | `ops/default/report/slices/object-storage/slice-1-local-objects.md` |
| Slice 2 | Private S3 objects | `feat/private-s3-storage` | `feat(slice-2): inspect private S3 objects` | `ops/default/report/slices/object-storage/slice-2-s3-objects.md` |
| Slice 3 | Signed POST uploads | `feat/signed-storage-uploads` | `feat(slice-3): sign constrained remote uploads` | `ops/default/report/slices/object-storage/slice-3-signed-uploads.md` |
| Slice 4 | Object storage guides | `docs/object-storage-workflows` | `docs(slice-4): guide private object storage workflows` | `ops/default/report/slices/object-storage/slice-4-storage-guides.md` |
| Slice 5 | Integrated acceptance | `test/object-storage-acceptance` | `test(slice-5): verify object storage acceptance` | `ops/default/report/slices/object-storage/slice-5-integrated-acceptance.md` |

## Tasks and Expected Commits

| Task | Deliverable and evidence responsibility | Expected commit |
| --- | --- | --- |
| T1.1 | Add the approved storage value/options/error contracts with a working local adapter: canonical keys, bounded reads/writes, metadata, atomic replacement and classified absence/uncertainty. Preserve caller reader ownership and return complete objects only. | `feat(storage): implement bounded local object operations` |
| T1.2 | Prove actual filesystem confinement, symlink replacement races, old-content preservation, real byte limits, blocking-reader boundaries, cancellation and owned spool cleanup. Create the concern runner with only delivered mode 1. | `test(storage): verify local ownership and resource bounds` |
| T2.1 | Add storage/s3 through the existing SDK, private configuration and typed options. Deliver Put/Open/Stat/Delete, response metadata checks, classified provider failures and bounded replayable input. Bind the pinned actual server fixture and binary identity. | `feat(storage): implement private S3 object operations` |
| T2.2 | Exercise actual remote round trips, confirmed absence versus denial, input/metadata bounds, observed snapshot identity, cancellation and unknown mutation outcomes. Extend runner mode 2 and retain actual fixture cleanup evidence. | `test(storage): verify private S3 transport outcomes` |
| T3.1 | Add explicit UploadSigner capability with exact bucket/key/type, content-length range and finite effective deadline. Bind SDK signing time/policy expiry and expiring credentials without publishing secret signing material or weakening unsupported providers. | `feat(storage): sign exact-target bounded POST uploads` |
| T3.2 | Submit real multipart forms; prove altered key/type, policy/signature, under/oversize and expiry rejection, allowed reuse, in-flight boundary and safe staging-to-distinct-final snapshot use. Extend runner mode 3 and preserve reader/error semantics. | `test(storage): verify signed upload enforcement` |
| T4.1 | Deliver linked tutorial, private-storage/signing/reconciliation how-to guides, exact reference and bearer/snapshot explanation. Compile examples and preserve all historical image constructors, key, URL and processor behavior. | `docs(storage): explain private object storage contracts` |
| T4.2 | Execute the documented local and actual signed-remote workflows in owned fixtures; detect API/configuration/example drift and secret logging. Extend runner mode 4 with meaningful documentation and image compatibility evidence. | `test(storage): execute documented storage workflows` |
| T5.1 | Compose lifecycle/error/compatibility and signed-remote evidence into OS01-OS10; introduce immutable acceptance ledger and canonical report with actual source/fixture identity. | `test(storage): verify integrated object lifecycles` |
| T5.2 | Enable full concern mode, complete root/command/result manifests and candidate/log/fixture/cleanup receipts. Reject missing or skipped required roots, drift and incomplete cleanup; require the existing full repository gate. | `test(storage): bind exact-candidate acceptance evidence` |

## Incremental Safety and API Bindings

Every introducing adapter delivers its cancellation, bounds, cleanup and error
semantics. Do not postpone critical byte enforcement or spool cleanup to Slice 5.
Each slice compiles and exposes working behavior; the local store never pretends
to sign, and the remote adapter does not acquire signing until Slice 3. Unknown
or unsupported capabilities fail explicitly.

T1.1 binds the specification's value fields and exact error predicates to
exported declarations and Godoc. T2.1 defines explicit failable construction,
private endpoint/bucket/credential configuration, spool policy and finite SDK
retry settings. T3.1 binds expiry to the actual signed policy rather than an
unchecked independent timestamp calculation. Neutral packages import no SDK;
adapter-owned code performs typed SDK mappings.

Keep existing image APIs intact. Sharing an internal implementation is permitted
only when regression evidence preserves original constructor, URL, accepted key,
error and image-size behavior. Do not force a caller migration merely to avoid
a small explicit adapter boundary. General metadata is not a validated MIME
promise, an integrity hash or a durable application record.

Update CHANGELOG Unreleased only when a slice delivers a usable package outcome,
using the changelog skill. No API-only promise for an unavailable implementation
or version/tag change belongs in these slices.

## Actual Remote Fixture

Reference server: MinIO official release `RELEASE.2025-10-15T17-29-55Z`.
Release source: [Official release](https://github.com/minio/minio/releases/tag/RELEASE.2025-10-15T17-29-55Z).
Immutable source: `9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`.
Resolved module: `github.com/minio/minio v0.0.0-20251015172955-9e49d5e7a648`.
These identities were checked against the official tag and Go module metadata
on 2026-10-16. The release is source-distributed; do not assume a published
container image, use an older unreviewed image or resolve a floating latest tag.

T2.1 provides an explicit Go/shell fixture-tool preparation command using this
exact module version and Go 1.27.1. Install its binary only into an ignored owned
tools directory, independently of the product module. Record module checksum,
Go build information and binary SHA-256 before starting remote tests. Preparation
may reuse only an exact matching executable. A different source/build requires
recorded plan reconciliation; no product dependency or system-wide install is
implied. This plan pins source identity, not a fabricated unbuilt binary hash.

The runner starts one owned native server with private scratch data, ephemeral
test credentials, loopback listeners and a uniquely named test bucket. Startup
and readiness are bounded to 60 seconds, transport effects to 120 seconds and
server teardown to 30 seconds. Tool compilation is a separate visible phase
with a 15-minute bound; never conceal it inside an apparently idle root.
Tests retain object counts and confirmed teardown without credentials, form
fields or private data in receipts. No already-running user server is stopped.
The test endpoint permits plaintext only under the explicit fixture option.

Use an actual server for policy enforcement, private access and snapshot tests.
Fault HTTP endpoints complement it for malformed replies and lost acknowledgments;
they do not replace it. Tests prove this identified server's compatibility, not
all S3-compatible vendors. External account operations require separate explicit
authority and are unnecessary for this set's acceptance.

## Focused Validation Contract

T1.2 creates `scripts/check-object-storage.sh`. It admits delivered `slice N`
modes only and rejects full mode until T5.2. Every canonical PR has this exact
ordered prefix, followed by extra successful focused commands if needed:

`GOWORK=off scripts/check-object-storage.sh slice N`

`git diff --check` is last. The runner checks Go 1.27.1, formatting, source
licensing, shell syntax, affected ordinary/race tests, affected build/vet/lint
and actual boundary results. T4 additionally requires `make docs-check` through
the runner. It emits root/fixture start and finish events with retained logs.
All following selectors and commands are future required outputs, not checks
that exist or passed during planning.

| Slice | New required roots | Existing targeted regressions |
| --- | --- | --- |
| 1 | storage ordinary: TestKeys, TestBounds; storage/local ordinary: TestObjects, TestConfinement, TestWriteFailure | All current image/local tests |
| 2 | storage/s3 integration: TestPrivateObjects, TestRemoteBounds, TestRemoteFailures | All current image/s3 and image/local tests |
| 3 | storage/s3 integration: TestSignedUploads, TestSigningExpiry, TestSnapshotCompletion | Introduced local/remote I/O and existing image/s3 tests |
| 4 | scripts/documentation-conformance ordinary: TestObjectGuideDrift; integration,documentation: TestObjectStorageGuide | All current image/... tests and compiled examples |
| 5 | storage/s3 integration: TestObjectLifecycle; scripts/documentation-conformance ordinary: TestObjectAcceptance | Complete current repository suite plus all introduced tagged/guide roots |

Short package paths resolve under the module root. Required integration roots
use `-tags=integration`, documentation roots `-tags=integration,documentation`.
Use `-json -race -count=1 -p=1 -parallel=2` for tagged root execution. Bound roots
to 360 seconds, guides to 600 seconds and record any evidence-based bound change.
Discover exact package/tag/root identity; a missing root, empty selection,
required skip, race or failed cleanup fails its slice. Preserve useful table
cases and concise test names; do not mirror implementation with low-signal tests.

## Full Gate, Evidence and Closure

After five canonical merges, the configured full command is:

`GOWORK=off scripts/check-object-storage.sh`

Full mode prepares exact owned fixture tools and executes `make check`,
`make docs-check`, the complete introduced integration/documentation roots and
their actual cleanup. The existing complete repository suite is not replaced
with storage-only tests. `make check` must leave the exact candidate unchanged;
formatting mutations fail receipt binding. Do not run `make ci` or rewrite badges.
Existing opt-in unrelated live-provider tests retain their declared boundaries;
record skips explicitly and never treat them as storage acceptance. Every
required OS root must execute without skip.

Full-mode PostgreSQL uses a fresh private native cluster through the existing
`db` tests' `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` contract. Supply
only the owned cluster to the gate process; do not use a system database or
require Docker socket access. T1.2 records the initdb/pg_ctl/psql prerequisite
and T5.2 wires bounded cluster startup and confirmed shutdown into the complete
runner. Other test-container fallbacks are not silently skipped. This is fixture
composition through existing test inputs, not a product database-contract change.

Receipts bind exact pushed dev candidate, ordered commands, Go/SDK identities,
complete manifests/results, source/build/binary fixture identity, log hashes,
OS01-OS10 mapping and teardown. Acceptance requires confirmed owned reader,
spool, server, bucket and temporary-root cleanup. No repair budget or silent
retry is granted by this plan; a failed full gate preserves diagnostics and
requires explicit correction/retry authority.

At activation, transition the source ticket open -> ready -> in_progress with
the required timestamps and matching source links. T5.2 moves it to reviewing
with all task/PR identities before the final gate; failed gate evidence does not
resolve it. Keep its original filename and ID through every directory move.

After the exact green candidate, restrict closure to these eleven path names;
the final ticket move deletes its reviewing path and creates its solved path:

- `ops/default/spec/object-storage.md`
- `ops/default/plan/object-storage.md`
- `ops/default/tracker/object-storage.md`
- `ops/default/report/object-storage-acceptance.md`
- `ops/default/report/slices/object-storage/slice-1-local-objects.md`
- `ops/default/report/slices/object-storage/slice-2-s3-objects.md`
- `ops/default/report/slices/object-storage/slice-3-signed-uploads.md`
- `ops/default/report/slices/object-storage/slice-4-storage-guides.md`
- `ops/default/report/slices/object-storage/slice-5-integrated-acceptance.md`
- `ops/default/ticket/reviewing/20261016081501-support-bounded-object-storage-and-signed-uploads.md`
- `ops/default/ticket/solved/20261016081501-support-bounded-object-storage-and-signed-uploads.md`

Preserve source, tests, dependency files, scripts, product docs, examples and
changelog from the green candidate. Resolve the source ticket with normal
reviewing/solved lifecycle metadata only after implemented acceptance; record
and validate its transitions rather than skipping state requirements. Update
links to its solved path during the same restricted closure.

Each report has exactly the canonical seven sections and retains its immutable
introduction. PRs use Summary, Changes and Validation with the immutable report
link last under Changes. Record significant outcomes in Acta. Closure does not
create a version, publish a module or promote main; those remain separately
authorized operations.

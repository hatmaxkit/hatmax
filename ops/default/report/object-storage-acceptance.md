<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Acceptance

Status: Completed
Completed: 2026-10-16
Validated candidate: `c03a4f042507eb8e255571aa2781fcf7a42f0036`
Delivery set: object-storage
Plan: [Delivery plan](../plan/object-storage.md)
Tracker: [Delivery tracker](../tracker/object-storage.md)
Source ticket: [Bounded objects and signed uploads](../ticket/solved/20261016081501-support-bounded-object-storage-and-signed-uploads.md)

## Acceptance Contract

The [immutable ledger introduction](https://forge.adrianpk.com/hatmax/hatmax/src/commit/9cde0688161bab2a69ef6b9a49cb62054bef5ae6/scripts/object-storage/ledger.go)
maps OS01-OS10 to current
package/tag/root identities. Runtime evidence binds that ledger, the candidate,
tracked source bytes, exact ordered command arguments, discovered root manifests,
results and log hashes. Source discovery or a slice receipt alone cannot accept
OS10. The controller owns the full `GOWORK=off scripts/check-object-storage.sh`
execution after five canonical merges; its exact retained evidence owns closure.

| Criterion | Observable evidence owners | Result |
| --- | --- | --- |
| OS01 | Local/remote document lifecycles and integrated TestObjectLifecycle | passed |
| OS02 | Canonical keys and actual directory-handle confinement/races | passed |
| OS03 | Actual input/output/metadata bounds, cancellation and ownership | passed |
| OS04 | Absence, denial, transport errors and indeterminate mutation reconciliation | passed |
| OS05 | Actual signed POST target/type/range/signature/expiry enforcement | passed |
| OS06 | Actual policy deadlines, credential boundaries, reuse and in-flight completion | passed |
| OS07 | One complete opened snapshot committed unchanged under a distinct final key | passed |
| OS08 | Historical image constructors, accepted keys, URLs and processor regressions | passed |
| OS09 | Exact published Go, compiled examples, local and actual signed guide execution | passed |
| OS10 | Exact integrated candidate, full repository gate, manifests and confirmed cleanup | passed |

## Fixture Identity

Native MinIO source `9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`, module
`github.com/minio/minio v0.0.0-20251015172955-9e49d5e7a648`, module checksum
`h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM=`, Go 1.27.1.
The observed executable SHA-256 from delivered slices is
`726b305d2753cc6b5585028b006bde0e6e7f58d7774b1308bc8b4eaa49eb5d6c`;
each execution verifies actual source/build/binary identity before startup.
Private loopback listeners, ephemeral credentials and uniquely owned buckets
supply acceptance. Production accounts and already-running services are unused.

The full gate provisions a fresh private native PostgreSQL cluster through
existing DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME test inputs. It records tool
identity, bounded startup and confirmed shutdown without logging credentials.
All required storage roots must run without skip. Existing unrelated opt-in
boundaries are recorded separately and cannot count as storage acceptance.

The runner emits an exact-candidate manifest, 21 ordered command/result records
and a hashed acceptance receipt under ignored `.tmp/object-storage/`. Discovery
uses actual build-tag-selected Go files, including runnable tests, examples and
fuzz seeds. Ordinary and race results, every introduced tagged root, sanitized
guide-child results and native cleanup are mandatory. Only full mode on clean
pushed dev, including the existing complete make check target, can accept OS10.
Slice 5 mode reports OS10 pending even when its focused evidence is green.

Full make check receives optional TEST_FLAGS for JSON, fresh execution and finite
scheduling. Its original license, format, vet, complete tests, 80% coverage and
strict lint prerequisites remain intact. PostgreSQL initialization/start/shutdown
are bounded to 60/30/30 seconds; its command has a 30-minute ceiling. Tagged
storage roots retain 360 seconds and the guide 600 seconds. No full gate is
executed during Slice 5 preparation.

## Final Gate Evidence

The controller passed `GOWORK=off scripts/check-object-storage.sh` on clean
pushed dev candidate `c03a4f042507eb8e255571aa2781fcf7a42f0036`.
Validation `ORAZRFW4UDUF4TSQEPDZ47HPFF` finished successfully on 2026-10-16.
The accepted receipt reports all OS01-OS10 passed. This closure changes only
approved documentation and the ticket lifecycle; the tested implementation is
preserved. The full command was not rerun for closure.

The receipt contains 21 successful ordered steps, 81 concern roots and a
1,175-root complete repository manifest. The existing make check gate passed
license, formatting, vet, complete tests, coverage and strict lint. Total coverage
was 80.9%, exceeding the unchanged 80% requirement. Required ordinary/race,
actual remote, guide and guide-child roots passed without skip. The two retained
unrelated skips are `generator/backend/codex:TestAuthenticatedLiveSmoke` for its
explicit external authentication opt-in, and `scheduler:TestPanicProcess` for the
subprocess helper exercised by its parent test. Neither satisfies an OS root.

The actual remote fixture cleaned eight objects to zero and removed its primary
and scope-probe buckets. The signed guide verified exact final bytes, empty
spools and zero remaining objects. Both owned MinIO servers and temporary roots
were removed. The private PostgreSQL 18.6 cluster confirmed readiness and shutdown,
verified tool/log hashes and removed its socket/data root. No owned fixture
process remains.

Retained evidence paths and SHA-256 values follow. The bounded controller log
ends with its 1 MiB diagnostic truncation marker; the complete repository command
log below retains the full gate output and coverage result.

| Artifact | Retained path | SHA-256 |
| --- | --- | --- |
| Controller receipt | `/home/adrian/.local/state/looper/hatmax-object-storage.json.state/checks/c03a4f042507eb8e255571aa2781fcf7a42f0036.json` | `fdf776ae15162902abdf0bdcd57fd8f516b0aa5065730f4d48cb870f16eb8ee9` |
| Controller log | `/home/adrian/.local/state/looper/hatmax-object-storage.json.state/checks/c03a4f042507eb8e255571aa2781fcf7a42f0036.ORAZRFW4UDUF4TSQEPDZ47HPFF.log` | `921b56e59eac3a88f8141884dc3ead895522300ce4fce7fd2c27423540c19aa8` |
| Candidate/root manifest | `/home/adrian/Projects/hatmax/hatmax/.tmp/object-storage/full.YzJJ2c0x/manifest.json` | `fea2428d23f7b5fcbecc29271efc38f43b731883fee0379040eae69257e65ad8` |
| OS01-OS10 acceptance | `/home/adrian/Projects/hatmax/hatmax/.tmp/object-storage/full.YzJJ2c0x/acceptance.json` | `4d399cabced2ecd67e6b91211516400aada02d6b4005eb7fc9e48786073074a8` |
| Complete repository gate log | `/home/adrian/Projects/hatmax/hatmax/.tmp/object-storage/full.YzJJ2c0x/check-18.log` | `809012caedbe142244c2cb2cee847fa6187683bacac6d853d2081e1ac555cc88` |

The acceptance receipt also hashes every step log, remote/guide cleanup receipt,
sanitized child-result stream, PostgreSQL receipt and tool-preparation log.
The candidate manifest records Go `go1.27.1`, AWS SDK `v1.47.1`, S3 SDK
`v1.114.0`, source/tree and evidence executable identities.

<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Acceptance

Status: Awaiting exact integrated gate
Delivery set: object-storage
Plan: [Delivery plan](../plan/object-storage.md)
Tracker: [Delivery tracker](../tracker/object-storage.md)
Source ticket: [Bounded objects and signed uploads](../ticket/reviewing/20261016081501-support-bounded-object-storage-and-signed-uploads.md)

## Acceptance Contract

The [immutable ledger introduction](https://forge.adrianpk.com/hatmax/hatmax/src/commit/9cde0688161bab2a69ef6b9a49cb62054bef5ae6/scripts/object-storage/ledger.go)
maps OS01-OS10 to current
package/tag/root identities. Runtime evidence binds that ledger, the candidate,
tracked source bytes, exact ordered command arguments, discovered root manifests,
results and log hashes. Source discovery or a slice receipt alone cannot accept
OS10. The controller owns the full `GOWORK=off scripts/check-object-storage.sh`
execution after five canonical merges; its exact retained evidence owns closure.

| Criterion | Observable evidence owners |
| --- | --- |
| OS01 | Local/remote document lifecycles and integrated TestObjectLifecycle |
| OS02 | Canonical keys and actual directory-handle confinement/races |
| OS03 | Actual input/output/metadata bounds, cancellation and ownership |
| OS04 | Absence, denial, transport errors and indeterminate mutation reconciliation |
| OS05 | Actual signed POST target/type/range/signature/expiry enforcement |
| OS06 | Actual policy deadlines, credential boundaries, reuse and in-flight completion |
| OS07 | One complete opened snapshot committed unchanged under a distinct final key |
| OS08 | Historical image constructors, accepted keys, URLs and processor regressions |
| OS09 | Exact published Go, compiled examples, local and actual signed guide execution |
| OS10 | Exact integrated candidate, full repository gate, manifests and confirmed cleanup |

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

Pending. No complete integrated gate has run for this delivery set yet.
Record the controller's exact candidate, command, receipt/log identities,
OS01-OS10 results and owned resource cleanup here only during authorized closure.

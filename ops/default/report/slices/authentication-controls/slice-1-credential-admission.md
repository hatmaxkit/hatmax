<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Durable Credential Admission

Status: reviewing
Delivery set: authentication-controls
Plan: `ops/default/plan/authentication-controls.md`
Tracker: `ops/default/tracker/authentication-controls.md`
Branch: `feat/credential-admission`
PR: `#107`

## Purpose

Add callable shared admission before credential work and prove its durable finite
boundaries. This is Slice 1 of 5 in the [plan](../../../plan/authentication-controls.md)
and [tracker](../../../tracker/authentication-controls.md). Mandatory password
entrypoint wiring belongs to Slice 2; this layer does not close AUTH-07.

## Delivered Behavior

- Registration and password proof have independent durable windows under one
  finite namespace capacity. Known and unknown identities require no user row.
- The final allowed operation commits its fixed cooldown. Denial changes no
  counters or times. Accepted operations have no refund or release API.
- Restart and looser settings retain captured limits and spent counts. Tighter
  settings cannot truncate spent counts. Renewal requires both window and
  cooldown retirement; exact equality permits renewal.
- A stable application-owned key produces private HMAC identity keys. Replicas
  share namespace/key; a different key for an existing namespace fails closed.
- Earlier caller deadlines bound lock waits. Explicit cleanup removes at most
  one configured batch, preserves live state and updates capacity atomically.

## Implementation Notes

T1.1: `f32729cd06e0`; T1.2: `c1ed78985aba`.

Core snapshots validated settings and checks identity structure without inventing
normalization. The PostgreSQL adapter locks namespace capacity before identity,
reads actual database time after locks and commits each admitted charge once.
It performs no account lookup, password verification, subject lock, callback or
background work. Failure after row insertion/update rolls back its capacity
change. Cleanup uses the same lock order and rechecks complete retirement.

Migration 009, typed queries, generated models and configuration match the
settled companion model. The adapter copies its bounded private key. Provision
stable namespace/key and consistent replica settings outside requests; changing
keys is an operating failure, not a budget reset. Ambiguous commits must not be
replayed automatically. Namespace-wide serialization makes the capacity invariant
explicit; measurements below do not establish deployment throughput.

## Contracts Added or Changed

`CredentialAdmissionQueries`, `CredentialAdmission`, closed registration/proof
purposes and classified attempt/capacity/state errors are new. `Admit` grants no
proof or session. Internal retry time must not reach unauthenticated responses.
`Cleanup` is caller-owned and has no worker or automatic retry.

`config.CredentialAdmissionConfig` resolves finite defaults: ten proof operations
per ten-minute window/cooldown, three registrations per one-hour window/cooldown,
10000 shared records, one-second operation timeout and 1000-record cleanup batch.
Invalid explicit settings and unresolved direct adapter snapshots are rejected.

`NewCredentialAdmissionStore` implements the shared PostgreSQL contract. Private
identity/binding digests are neither authentication bearers nor event fields.
Existing password paths and factor/recovery admission are preserved in this slice.

## Files of Interest

- `auth/credential_admission.go`: typed admission, structural checks and caller deadlines.
- `config/credential_admission.go`: finite defaults and validated snapshots.
- `examples/ticked/internal/feat/auth/credential_admission.go`: durable charge and cleanup transactions.
- `examples/ticked/assets/migration/postgres/009-credential-admission.sql`: private counters, shared capacity and retirement index.
- `examples/ticked/internal/feat/auth/credential_admission_integration_test.go`: actual races, fault rollback, deadlines and storage measurement.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#standalone-credential-admission)
  and [configuration](../../../../../docs/reference/configuration/README.md#credential-admission-limits).

## Validation

Focused checks passed with Go 1.27.1 and owned PostgreSQL 18.6. Database tests
require explicit owned connection settings, isolate/drop schemas and generate
fixture keys without logging them. The owned database was stopped after all
checks and a zero-other-client check. No external provider, browser or production
validation was used. The full integrated gate remains after five canonical merges.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed: zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -v -run '^TestCredentialAdmissionTransactions$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 5.642s.
- `go test -run '^$' -fuzz '^FuzzCredentialIdentity$' -fuzztime=20s -parallel=2 -timeout=60s ./auth` — passed: 398490 executions, no KDF/database work.
- `git diff --check` — passed.

Two actual instances admitted exactly ten of 24 simultaneous proof operations.
The selector also verifies independent registration, restart/key binding, policy
tightening without count mutation, trusted window/cooldown retirement, finite
capacity, bounded cleanup, cleanup/renewal races and trigger-induced rollback
for charge and cleanup. A held capacity lock respects an 80ms caller deadline
without admitting a new row. Exact equality and one-microsecond boundaries use
the production window transition helper; distributed semantics use real SQL.

The owned fixture inserted 1000 admissions in 2.864143392s. Its admission table
used 122880 heap bytes and 212992 index bytes. This is a bounded fixture cost
measurement, not a production load or side-channel audit. Core fakes cover only
orchestration, invalid structure and late/canceled adapter results.

## Risks and Follow-ups

- Slice 2 must wire this shared contract before all actual password entry work.
  Public ingress, neutral responses, observations and browser evidence remain
  in the approved slices; no usable guarded authentication claim is made here.
- Applications own canonical identity/alias convergence, stable namespace/key,
  consistent replica configuration and bounded cleanup scheduling. A lowered
  capacity retains existing rows; new identities wait for available occupancy.
- Temporary fixed-window denial does not implement cumulative authenticator
  disabling/rebinding, all-factor-loss proofing or assurance certification.

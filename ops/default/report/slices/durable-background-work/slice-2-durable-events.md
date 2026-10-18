<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Deliver bounded durable events

Status: delivered
Delivery set: durable-background-work
Plan: [Delivery plan](../../../plan/durable-background-work.md)
Tracker: [Delivery tracker](../../../tracker/durable-background-work.md)
Branch: `feat/bounded-durable-pubsub`
PR: `#130`

## Purpose

Publish bounded events transactionally with domain state and recover independent
subscriber attempts within retained budgets, without losing late commits or
starving later work behind terminal failures.

## Delivered Behavior

- Explicit topic admission validates identity, JSON, metadata, topic/subscriber
  counts and retained capacity. Canonical exact publication retries are idempotent;
  conflicting reuse and exhausted capacity return classified errors.
- `PublishTx` leaves transaction ownership with its caller. Capacity slots roll
  back with publication and permit independent transactions to commit out of ID
  order. An exact duplicate succeeds at full capacity without another reservation.
- Named subscribers choose current-only or retained-history admission. Each
  delivery freezes its policy on first claim and replaces expired ownership with
  a fresh token and increasing attempt. Stale acknowledgements cannot write state.
- Successful acknowledgement, diagnostic progress and terminal delivery state
  commit atomically. Errors, cooperative deadlines, panics and invalid retained
  data consume finite attempts; failed deliveries have no success acknowledgement
  and are excluded from polling so later messages can progress.
- Fetch results close before handlers run. Bounded subscriptions inherit their
  lifecycle context; `Stop(ctx)` cancels and bounds the wait without closing the
  caller-owned database or spawning replacement handlers.
- Explicit legacy topic/subscriber upgrades preserve successful acknowledgements
  and require an admission audit and selected policy/history. Existing legacy
  subscription calls on unadmitted topics keep their documented delivery behavior.

Merged into `dev` through controller-owned PR #130 at
`ebba2ebaf991d3b49681f835e2629b191059a4f2`; the required ordered Slice 2 checks
and whitespace check passed at that head.

## Implementation Notes

T2.1 is `ab080c841c2b372a1b6e51a7f2180448bc91b70d`.
T2.2 is `c52d6cb2eb8e7c9a6247fcee1faacb74037ae4b3` and supplies subscriber
policy/claims, safe failure handling, upgrade guidance and the Slice 2 gate mode.
Merged Slice 1 PR #129 is closed in its existing report/tracker entries.
The controller owns this slice's Rebase + Fast-forward integration into `dev`.

Admission serializes identical message IDs while different messages reserve
separate capacity slots. Operator admission changes serialize against publications.
Registration limits use a separate short transaction lock, leaving late-committing
publication eligible under the acknowledgement snapshot boundary. No scalar offset
is used as a delivery exclusion cursor. Fixed subscriber policy changes apply only
to future deliveries; existing snapshots retain attempt limits and retry delays.

## Contracts Added or Changed

`Config.Bounded`, `BoundedConfig`, `PublishTx`, `AdmitBoundedTopics`,
`SubscribeOptions.Durable`, `DurableOptions`, `DeliveryPolicy`, `HistoryMode` and
`ConfigureDurableSubscription` provide explicit bounded publication/subscription.
Identity, capacity, policy, admission and stale-ownership conflicts are classified.
The default policy is 10-second cooperative execution, 30-second lease, eight total
attempts and 30-second retry delay, within the approved bounds.

Additive topic/capacity/delivery tables and conditional message/subscription
constraints leave unrelated legacy rows unchanged. Legacy acknowledgements remain
successful. Bounded delivery payload numbers use `json.Number`. Retained failure
codes exclude raw handler, panic and provider text. Legacy polling/ack methods
cannot bypass bounded ownership.

## Files of Interest

- `pubsub/durable.go`: validated policy and explicit history admission.
- `pubsub/postgres/publication.go`: canonical identities, transaction publication and audit.
- `pubsub/postgres/subscription.go`: fixed registration policy and explicit upgrades.
- `pubsub/postgres/durable_delivery.go`: claims, frozen budgets and fenced acknowledgement.
- `pubsub/postgres/schema.go`: conditional additive schema and retained capacity.
- `pubsub/postgres/publication_test.go`, `durable_delivery_test.go`: real PostgreSQL boundaries.
- `docs/how-to/use-pubsub/README.md`: setup, transaction and upgrade procedures.
- `scripts/check-durable-background-work.sh`: independent Slice 2 mode.

## Validation

Passed focused local checks before the final controller receipt:

- `scripts/durable-background-work/postgres-fixture.sh .tmp/durable-background-work/focused env GOWORK=off go test -race -count=1 -p=2 -timeout=90s ./pubsub ./pubsub/postgres -run '^(TestBoundedDeliveries|TestBoundedPublication|TestLegacyDelivery|TestLateCommit|TestSubscriptionSnapshot|TestMigrationRollback)$'`
- `scripts/durable-background-work/postgres-fixture.sh .tmp/durable-background-work/focused env GOWORK=off go test -race -count=1 -p=2 -timeout=60s ./pubsub/postgres -run '^TestBoundedPublication/concurrent_duplicate$'`
- `GOWORK=off GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./pubsub ./pubsub/postgres`
- `GOWORK=off make docs-check`
- `bash -n scripts/check-durable-background-work.sh`
- `git diff --check`

Focused package race executions reported 1.013 s and 3.891 s. Owned native
PostgreSQL stopped and removed its temporary root. The final clean PR HEAD uses
the controller's ordered slice-command/whitespace receipt; these focused results
are not the delivery-set gate.

The controller passed `GOWORK=off scripts/check-durable-background-work.sh slice 2`
and `git diff --check` at `93bd762f32990cc22fed5ddb3d0f972574de1f20` before this
evidence update. Native measured wall times were:

| Selected command | Seconds |
| --- | --- |
| `GOWORK=off go vet ./pubsub ./pubsub/postgres` | 0.41 |
| `GOWORK=off go test -race -count=1 -p=2 -timeout=180s ./pubsub ./pubsub/postgres` | 7.49 |
| `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./pubsub ./pubsub/postgres` | 0.96 |
| `make docs-check` | 5.26 |

The owned native PostgreSQL fixture stopped and removed its temporary root. The
selected commands totaled 14.12 seconds, below the estimated warm-cache budget.
The controller regenerates the same ordered receipt after this report commit;
only evidence for the final clean PR HEAD authorizes integration.

BW02 (pubsub): competing brokers, an actually killed worker and stale-generation
acknowledgements. BW04: actual domain commit/rollback, canonical duplicate and
conflict tests. BW05: retained/current history, late commits, fan-out, fixed retries,
terminal failure quarantine and later progress. BW07 (admission): concurrent
capacity reservations, rollback, duplicate-at-capacity, input and registration
limits. BW08 (pubsub): classified diagnostics and explicit legacy admission upgrades.
Replay, retirement, fan-out-aware pruning and final integrated acceptance remain
with the recorded Slice 3 tasks.

## Risks and Follow-ups

Handler cancellation is cooperative and an expired owner can still perform an
external effect. Applications own deduplication and operator access. Retained
messages consume capacity until explicit fan-out-aware retention, completed by
Slice 3. Old workers must be quiesced before admission upgrades; invalid legacy
data is rejected rather than truncated or implicitly replayed.

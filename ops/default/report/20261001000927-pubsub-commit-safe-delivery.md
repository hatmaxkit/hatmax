<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Pubsub Commit-Safe Delivery

Status: reviewing
Ticket: [TKT-20260930211804](../ticket/reviewing/20260930211804-make-pubsub-cursors-safe-for-late-commits.md)
Branch: `fix/ticket-20260930211804-pubsub-commits`
PR: [#72](https://forge.adrianpk.com/hatmax/hatmax/pulls/72)
Implementation: `a061bda59b92403791d1f189bd8479bed0797fcc`

## Delivered Behavior

- Polling selects committed topic messages without a per-subscriber
  acknowledgement. A lower ID committed after a higher ID was delivered remains
  eligible, including after a named subscriber restarts.
- New subscriptions exclude only topic history visible in their registration
  statement's snapshot. In-flight lower IDs are not mistaken for processed history.
- Each subscriber has independent tracking for fan-out. Returned rows, handler
  calls, and the in-memory ID buffer stay within the configured batch size.
- Exact batch acknowledgements and diagnostic `last_message_id` progress commit
  in one atomic SQL statement. Failed persistence leaves the batch pending and
  does not advance its in-memory progress.
- Startup transactionally adds the acknowledgement schema and seeds legacy
  history once. Repeated startup cannot acknowledge newly committed low IDs.

## Contracts and Ownership

PostgreSQL sequence allocation is not transaction commit order; see the
[sequence contract](https://www.postgresql.org/docs/current/functions-sequence.html).
A scalar high-water mark therefore cannot establish that every lower ID has
already been delivered. The new ledger records the exact subscriber/message
pair instead. Publisher transactions are not serialized and no dependency was
added; the implementation uses the existing `lib/pq` dependency.

Registration uses one data-modifying CTE statement so the maximum visible ID and
history exclusion share a snapshot. Batch persistence also uses one modifying
statement so its acknowledgement insert and diagnostic update roll back together.
These boundaries follow PostgreSQL's
[statement snapshots](https://www.postgresql.org/docs/current/transaction-iso.html)
and [data-modifying CTE semantics](https://www.postgresql.org/docs/current/queries-with.html).

For a legacy subscriber, migration acknowledges committed rows visible at startup
whose topic matches and whose ID is at or below its old cursor. The initialization
marker prevents repeating that backfill. Already skipped rows below the legacy
cursor cannot be distinguished from processed history and are not automatically
recovered. Older cursor-based brokers must be stopped before upgrade; mixed
versions writing delivery state are not supported.

Acknowledgements consume one row per subscriber/message pair, including history
excluded when registering a new subscriber. Neither messages nor acknowledgements
have automatic retention cleanup. Batch limits do not bound total database scan
work, which can grow with retained history. Indexes cover `(topic, id)` and
`(subscriber_id, message_id)`; capacity and retention remain application-owned.
Message/subscription deletion cascades to its acknowledgement records. Named
subscriber IDs remain bound to their original topic.

The [Pubsub reference](../../../docs/reference/pubsub/README.md#delivery-tracking)
documents this contract, upgrade boundary, duplicate-delivery window, and storage
cost. `Unreleased` records the corrected late-commit behavior.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`. Database tests used an isolated native
PostgreSQL 18.6 cluster on loopback, stopped after validation; this is local
evidence, not a claim that the PostgreSQL 16 CI job ran.

- Before the fix, `go test ./pubsub/postgres -run '^TestLateCommit$' -count=1`
  failed in both named cases: the lower-ID transaction remained undispatched
  after the higher-ID row was delivered, with and without broker restart.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.1%; `pubsub/postgres`: 89.8%.
- `make docs-check`: passed.
- `go test ./pubsub/... -count=1`: passed.
- `go test ./pubsub/postgres -count=1 -timeout=60s`: passed.
- `go test -race ./pubsub/... -count=1 -timeout=60s`: passed.
- `go test -race ./pubsub/postgres -run '^(TestLateCommit|TestSubscriptionSnapshot)$' -count=50 -timeout=60s`:
  passed all 50 repetitions of late-commit and registration-snapshot regressions.
- `git diff --check`: passed.

Tests explicitly control transaction commit order without timing sleeps. They
cover same-broker delivery, named restart, visible-history exclusion, late rows
at registration, batch sizes 1/2/3, independent subscribers, topic isolation,
legacy migration, repeated startup, migration rollback, and atomic acknowledgement
rollback. Fault injection verifies that a failed diagnostic update cannot commit
its acknowledgement insert; redelivery after that failure is expected.

A local synthetic `EXPLAIN (ANALYZE, BUFFERS)` probe used 10,000 topic rows and
9,900 acknowledgements, returning the 100 pending rows in 3.039 ms. The planner
chose a hash anti-join scanning both retained sets. This illustrates the history
scan cost; it is not a throughput guarantee. The probe used temporary tables in
a rolled-back transaction and left no persistent data.

## Boundary

Only finding F5 is addressed. Handler errors and invalid payloads retain the
existing log-and-acknowledge behavior; failure retries remain F6. A process crash
or acknowledgement persistence failure after handler execution can still repeat
delivery, so handlers must tolerate duplicates. This change does not provide
exactly-once execution, recover previously lost legacy history, implement retention,
or impose transaction commit order on delivery. Public broker APIs are unchanged.

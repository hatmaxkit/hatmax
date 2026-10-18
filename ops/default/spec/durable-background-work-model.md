<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Durable Background Work Model

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Companion to: [Durable background work](durable-background-work.md)
Implementation status: Completed
Completed: 2026-10-18
Validated candidate: `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`
Acceptance: [Integrated gate result](../tracker/durable-background-work.md#gate-evidence-and-closure)

## Ownership and Shared Claim Value

Extend the existing scheduler and pubsub Postgres schemas in their own packages.
Do not create a shared domain-job table, event bus service or application receipt
registry. A claim is a value containing logical identity, attempt, generation token
and database-clock expiry. Claim tokens are opaque ownership proofs, not user/session
credentials; never expose them through public endpoints or routine diagnostics.

All deadlines use `timestamptz`; persisted attempts are positive integers and never
decrease. A token change fences the previous owner. Store the execution policy with
the logical work, not just in the process that currently happens to run it.

## Scheduled Jobs and Runs

Keep `scheduled_jobs` and the `UNIQUE(job_id, scheduled_for)` slot invariant.
Add a deliberate durable-policy representation to the job configuration and snapshot
it into the run when claimed. Existing recurring schedule JSON and timezone rules
remain authoritative; do not add a second schedule language.

The following fields extend `job_runs` in this order after its current retry fields:

| Field | Representation | Constraint |
| --- | --- | --- |
| Durable execution | `durable boolean NOT NULL DEFAULT false` | Legacy rows are not silently adopted |
| Handler limit | `handler_timeout_ms integer` | Durable rows: 1,000–600,000 |
| Lease limit | `lease_duration_ms integer` | At least handler limit + 5,000; at most 900,000 |
| Retry delay | `retry_delay_ms integer` | Durable rows: 1,000–3,600,000 |
| Ownership | `lease_token text` | Present only while running; opaque fresh token |
| Ownership expiry | `lease_expires_at timestamptz` | Present with token; same attempt's database-clock deadline |
| Last classified failure | `failure_code text` | At most 64 ASCII bytes; no arbitrary error text |

Existing `attempt`, `retry_limit`, `retry_at`, `status`, `finished_at` and `output`
retain their meanings. Durable retry limits are 1–32 total attempts and output JSON
is bounded to 8 KiB before persistence. New bounded failure paths leave the legacy
raw `error` field empty; old records are not rewritten to claim they were sanitized.
Add a partial index for durable active lease expiry and retain existing due/retry
indexes. Terminal runs have no ownership token or expiry.

Transitions:

```text
unclaimed slot --atomic claim--> running(token A, attempt 1)
running --owned error--> retry_wait --due claim--> running(token B, attempt + 1)
running --expired claim--> running(token C, attempt + 1)
running/retry eligibility --budget exhausted--> failed
running --owned completion--> success
success/failed --same transaction--> recurrence advance or one-shot retirement
```

Expiry at exhaustion finalizes the slot transactionally. Old token holders cannot
complete, defer or update schedule state. Legacy store methods additionally require
`durable=false`; capability-specific writes match identity, attempt and token.

## Messages, Subscribers and Deliveries

Retain `pubsub_messages`, `pubsub_subscriptions` and successful
`pubsub_acknowledgements`. Transactional publication uses the same tables and the
same canonical identity validation as ordinary bounded publication. Store a digest
of canonical topic, payload JSON and metadata JSON to distinguish an exact duplicate
from conflicting reuse; JSON object key order alone is not a conflict. Enforce the
row capacity with an atomic admission boundary, including prune/rollback accounting.

Add `bounded boolean NOT NULL DEFAULT false` and nullable
`envelope_digest bytea` to messages. Bounded rows require a 32-byte digest and the
admitted payload/metadata sizes; the conditional constraints preserve legacy rows.
Bounded capacity counts bounded retained rows. A legacy row is not silently adopted
by a repeated publication; its topic must first pass the explicit admission upgrade.

The additive nullable `occurred_at timestamptz` retains the first non-zero envelope
timestamp independently of database `created_at`. Delivery uses
`COALESCE(occurred_at, created_at)` for zero timestamps and pre-upgrade rows. The
timestamp stays outside the identity digest; duplicate publication never replaces
the retained occurrence time. Retention and claim deadlines use database time.

Persist a bounded-mode flag, fixed execution policy and history-admission choice on
the subscription. Legacy rows retain their existing mode.
Reject reopening its name with a different topic or conflicting fixed policy.
An explicit upgrade is required to change the policy for future deliveries; existing
deliveries keep their original snapshot. Registration and publication must agree on
which committed and late-committing rows are eligible.

Add `pubsub_deliveries` with these ordered fields:

| Field | Representation | Constraint |
| --- | --- | --- |
| Subscriber | `subscriber_id text NOT NULL` | FK to subscriptions; part of primary key |
| Message | `message_id bigint NOT NULL` | FK to messages; part of primary key |
| State | `state text NOT NULL` | `running`, `retry_wait`, `succeeded`, `failed` |
| Attempt | `attempt integer NOT NULL` | Increasing, 1–32 |
| Total authorized budget | `attempt_limit integer NOT NULL` | 1–32; never below attempt |
| Handler limit | `handler_timeout_ms integer NOT NULL` | Same limits as scheduled work |
| Lease limit | `lease_duration_ms integer NOT NULL` | Same relationship to handler limit |
| Retry delay | `retry_delay_ms integer NOT NULL` | Same retry-delay limits |
| Ownership | `lease_token text` | Paired with lease expiry only in running state |
| Ownership expiry | `lease_expires_at timestamptz` | Database clock |
| Next attempt | `retry_at timestamptz` | Required only for retry_wait |
| Failure | `failure_code text` | Bounded classified code |
| Explicit replay time | `replayed_at timestamptz` | Last accepted replay, if any |
| Explicit replay reason | `replay_reason text` | Bounded documented reason code |
| Created, updated, terminal times | `created_at`, `updated_at`, `finished_at timestamptz` | Terminal time only for terminal states |

Primary key `(subscriber_id, message_id)` provides one logical delivery. Index
eligible retries and running lease expiry by subscriber; index terminal times for
bounded inspection/pruning. A claim inserts the first attempt or conditionally
updates the exact due/expired record. Concurrent losses are ordinary claim misses.

Successful completion inserts the existing acknowledgement and changes this row to
`succeeded` in one transaction. Failed/exhausted deliveries have no success ack.
The pending selector excludes all terminal deliveries as well as acknowledgements.
Do not use `last_message_id` as an exclusion cursor. A successful legacy ack requires
no synthetic attempt history and remains excluded from delivery.

Replay matches the observed terminal attempt and updated timestamp, adds a finite
budget within the lifetime maximum, clears terminal time and enters retry_wait.
It preserves attempt history counters and changes no successful domain receipt.
Retention deletes message/tracking rows together only after all entitled subscribers
are terminal and the caller cutoff is satisfied. Discarding a subscriber's pending
work is an explicit administrative action, never an automatic inactivity policy.

## Upgrade, Capacity and Stored Diagnostics

Validate new constraints against existing data through an additive upgrade. Do not
apply a `CHECK` that retroactively makes an unrelated legacy installation fail with
invented sanitized fields. Existing unbounded messages/subscribers need an explicit
admission audit before opting their topic into bounded mode; do not silently truncate
or delete them. Quiesced adoption of interrupted legacy runs records a real policy,
retains cumulative attempts and refuses terminal work or a newer current slot.

Capacity accounting must survive rollback, duplicate publication and pruning; it
cannot be a racy read-count-then-insert. Topic/subscriber registration is finite and
operator-owned. A blocked retention queue must surface capacity pressure rather than
grow without bound. No raw message copy or separate dead-letter payload table is
needed: failed delivery state refers to the retained original message.

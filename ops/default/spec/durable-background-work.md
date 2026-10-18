<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Durable Background Work

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Concern: durable-background-work
Model: [Durable work model](durable-background-work-model.md)
Plan: [Delivery plan](../plan/durable-background-work.md)
Tracker: [Delivery tracker](../tracker/durable-background-work.md)
Planning authorized: 2026-10-18
Inspected baseline: `b21af0b17c01ebb3544c37d7d200d2c87f4db0b1`
Owners: existing `scheduler`, `scheduler/postgres`, `pubsub` and `pubsub/postgres` packages
Implementation status: Completed
Completed: 2026-10-18
Validated candidate: `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`
Acceptance: [Integrated gate result](../tracker/durable-background-work.md#gate-evidence-and-closure)

## Purpose and Existing Behavior

Allow an application to recover interrupted scheduled work and operate durable
subscribers with finite attempts, explicit terminal failures and safe retention.
Reuse PostgreSQL, caller contexts and the existing package boundaries.

The inspected scheduler persists retry waits and enforces one run per scheduled
slot. A process can nevertheless leave a pending/running slot permanently claimed.
Completion has no ownership generation. Pubsub persists per-message acknowledgements
but has no cross-process delivery claim, total attempt budget or retention API.
Publication cannot currently share a caller's domain transaction. A failing message
can occupy a poll batch indefinitely. These are transport/lifecycle concerns
independent of an application's domain or reporting destination.

## Compatibility and Ownership

Introduce an explicit durable execution policy and fenced store capability rather
than silently requiring every existing custom store to implement a larger interface.
Existing scheduler constructors and store contracts remain usable without that policy.
Enabling durable execution against a store without the capability fails before work
starts. Legacy completion methods must reject leased rows, preventing an unfenced
writer from bypassing the new contract.

Bounded durable subscription is explicit and requires a stable subscriber identity.
Existing subscription calls retain their documented history/admission behavior.
An application must select bounded durable mode for the guarantees below. Do not
present legacy mode as interruption-safe or finitely retried.

Applications own transactionally protected effects, authorization and external
idempotency keys. Transport acknowledgement is not evidence of an exactly-once
business effect. No additional broker service or generic workflow engine is needed.

## Claims, Deadlines and Recovery

Claims use the PostgreSQL clock and an opaque generation token. Claiming an initial
run, a due retry or an expired attempt is atomic across replicas. Only the winning
token may complete, defer or fail the current attempt. Completion and schedule
advancement remain one transaction. A late worker receives a stale-claim result and
cannot change the replacement attempt or advance the schedule.

The default policy is a 10-second cooperative handler deadline, a 30-second lease,
eight total attempts and a fixed 30-second retry delay. Admit handler deadlines
from 1 second to 10 minutes, leases from deadline plus 5 seconds to 15 minutes,
attempt budgets 1–32 and retry delays 1 second–1 hour. Freeze the policy for the
run/delivery when first claimed; configuration changes do not reset existing budgets.
Do not introduce lease-heartbeat goroutines for these bounded attempts.

Lease expiry makes an interrupted attempt eligible for another attempt and consumes
the retained budget. It never implies that the old process has stopped or that a
remote side effect failed. Handler cancellation is cooperative; external effects
must use domain idempotency, and SQL effects must use a transactional deduplication
boundary. Do not spawn replacement goroutines to bypass an unreturned handler or
describe a context deadline as forceful termination.

Cancellation stops new claims and propagates into active work. Scheduler Stop and
Postgres broker Stop respect the caller's deadline without closing a caller-owned
database. A timeout reports outstanding work honestly; no process signals are sent.

## Scheduled Runs

In durable mode the initial claim creates a running attempt and its fixed policy
atomically, eliminating the gap between CreateRun and MarkRunning. Retry waits and
expired leases retain the same run/slot identity with a new token and increasing
attempt. Exhaustion records a terminal failed result and advances a recurring
schedule, or retires a one-shot job, as existing terminal completion does.

Preserve current recurrence semantics: advance from terminal completion and skip
missed slots. A crash does not create a new logical slot or an unlimited catch-up
backlog. Read-only inspection exposes active, retry-waiting, expired and terminal
runs, with their bounded failure code, attempts and next eligible time.

## Transactional Publication and Durable Delivery

Add an explicit Postgres transaction publication operation accepting a caller-owned
`*sql.Tx`. It never commits or rolls back that transaction. Domain state and its
message therefore become visible together. Validate one canonical message identity;
an exact repeated publication is idempotent, while reuse with different topic or
payload/metadata is a conflict. Existing Publish uses the same admission rules for
the new bounded mode and reports admission/persistence errors to the caller.

The bounded mode admits topics and named subscriber identities of 1–128 ASCII bytes,
message IDs of 1–128 ASCII bytes, JSON payloads up to 64 KiB and metadata up to 8 KiB.
Identity characters are letters, digits and `_.:/-`, with an alphanumeric first
character; control bytes and arbitrary display text are rejected. It supports at
most 128 configured topics and 64 durable subscribers. Its default
retained-message capacity is 10,000 rows, configurable from 100 to 100,000. Admission
enforces capacity atomically across concurrent publications, including caller-owned
transactions. Full capacity returns an explicit error; it never drops older pending
work or acknowledges a failed publication. Document the storage bound: at the default
maximum payload, payload data alone can approach 625 MiB, plus indexes/delivery rows.

Close finite result sets before invoking a handler; do not hold a fetch transaction
or database connection across arbitrary handler/network work. Each named subscriber
claims independently, preserving fan-out. Successful handling
records its acknowledgement atomically with terminal delivery state. Errors or
expired claims enter retry wait until their fixed budget is exhausted, then become
`failed`. Failed deliveries remain inspectable and are excluded from polling; later
messages remain eligible. A failed delivery is not represented as a successful ack.
Invalid payload/metadata is a classified failed attempt in bounded mode.

Registration explicitly chooses current-publication-only or retained-history
admission. Preserve the late-commit guarantee: a lower ID committing later cannot
disappear behind a scalar offset. Never promise global commit order or strict
processing order. A durable subscriber name remains bound to its original topic.

An explicit replay operation targets one terminal delivery and its observed revision.
It preserves the cumulative attempt count and grants at most eight additional
attempts, within the lifetime maximum of 32. Replays retain a bounded reason code
and replay timestamp. Already successful, active, missing or concurrently changed
deliveries cannot be replayed. Domain handlers must still reject obsolete effects.

## Inspection, Failure Codes and Retention

Expose bounded backend operations for listing work, replaying eligible failed
deliveries and pruning terminal records. Inspection defaults to 50 rows, allows at
most 100 and uses stable keyset pagination. Include task/topic/subscriber identity,
state, attempts, relevant timestamps and a classified code, without dumping message
payloads, SQL errors or arbitrary handler text. Applications decide operator access.

Pruning is explicit, caller-owned work, not a hidden service. A batch removes at
most 1,000 eligible rows within a caller deadline. Terminal scheduler runs may expire
after the caller-selected cutoff. A message may expire only when every registered
subscriber entitled to it is terminal and the caller's retention cutoff has passed;
delete associated tracking rows in the same transaction. Pending, active and retrying
work remains. Retiring a subscriber requires an explicit operation and cannot silently
discard its pending deliveries. Retention is an eligibility rule, not an assertion
that all expired rows disappear in one batch.

Bounded mode stores failure codes of at most 64 ASCII bytes and bounded result JSON
of at most 8 KiB. It does not persist raw panic values, payloads as diagnostics or
arbitrary SQL/provider error strings. Capacity/replay/claim conflicts have explicit
classified outcomes that a caller can log or report safely.

## Migration and Delivery Evidence

The model owns additive schema upgrades, indexes and validation of existing rows.
Stop old workers before installing new ownership semantics; mixed unfenced and
fenced executables are unsupported. Existing acknowledgements remain successful.
Legacy runs are not automatically declared abandoned or assigned invented budgets.
Provide an explicit quiesced adoption procedure for inspecting and adopting eligible
interrupted legacy runs. Do not replay historical messages implicitly.

Exercise the contracts against owned real PostgreSQL instances, including two
workers, process interruption and stale completion. Use short controlled deadlines
and actual durable state, not production-length sleeps. Reuse existing fixtures and
batch affected-package race tests; choose the final gate in the delivery plan.
Do not inherit unrelated repository suites or create a receipt-verification framework.
Document setup, API behavior, upgrade/recovery and operational guarantees in the
existing four documentation quadrants.

## Acceptance

| ID | Observable acceptance |
| --- | --- |
| BW01 | Existing store/constructor behavior remains supported; unsupported durable capability fails before execution and unfenced writes cannot alter leased rows. |
| BW02 | Two processes cannot own the same current attempt; a killed worker's eligible work resumes within the retained budget and stale completion is rejected. |
| BW03 | Retry waits survive restarts; exhaustion advances/retires the slot once; caller cancellation and shutdown deadlines remain effective. |
| BW04 | Domain rollback publishes no message; commit publishes one; exact repeated identity is idempotent and conflicting reuse fails. |
| BW05 | Independent durable subscribers retain late commits, claim with finite attempts and quarantine failures without starving later messages. |
| BW06 | A stale replay request cannot reset counters or repeat acknowledged work; admitted replay has an explicit finite remaining budget. |
| BW07 | Concurrent admission respects message capacity and input limits; pruning preserves unfinished work and the correct fan-out history. |
| BW08 | Inspection is bounded and excludes unsafe diagnostic data; migration/recovery guidance and real PostgreSQL evidence match the documented guarantees. |

The specification/model and all three slices/six tasks are complete. The controller
passed the selected integrated gate at `337f4666714f9bdc3a1e08b3cd0cf33643ddcd1a`
on 2026-10-18, accepting BW01–BW08 with owned PostgreSQL cleanup. The linked tracker
records the immutable evidence, task/PR references and canonical Go module identity.

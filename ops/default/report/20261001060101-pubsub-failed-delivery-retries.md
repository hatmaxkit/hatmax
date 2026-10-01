<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Pubsub Failed-Delivery Retries

Status: delivered
Ticket: [TKT-20260930211805](../ticket/solved/20260930211805-retain-failed-pubsub-deliveries-for-retry.md)
Branch: `fix/ticket-20260930211805-pubsub-retry`
PR: [#73](https://forge.adrianpk.com/hatmax/hatmax/pulls/73)
Implementation: `6d7aacf71c15e9ac95f081de1dd831b4803a4399`
Integrated into `dev`: `f7cfb4232e69f233909b9658297408d824b647c9`

## Delivered Behavior

- A handler error leaves its message pending instead of acknowledging it.
  Invalid JSON payloads also remain pending without invoking a handler.
- Failure does not abort the batch. Later selected messages can succeed and
  receive exact acknowledgements without acknowledging the failed row.
- Named subscribers retry pending failures after restart without repeating
  acknowledged successes. A higher diagnostic offset cannot hide a failed row.
- Each selected message gets at most one handler attempt per poll, within
  `BatchSize`. A resettable timer waits `PollInterval` before the first poll and
  after each completed cycle, including processing and persistence. Slow handlers
  cannot accumulate a ready ticker event for an immediate retry.
- `Close` cancels and waits for the loop. Context-aware handlers can stop a
  pending attempt without acknowledging it.

## Contracts and Ownership

F5's exact acknowledgement ledger is unchanged. Only successful handler returns
enter the batch's acknowledgement insert and diagnostic progress update; that
SQL statement remains atomic. No schema, public signatures, configuration fields,
or dependencies were added. Polling cadence now includes a post-completion delay,
rather than a fixed ticker period that could leave a queued tick after slow work.

The failure policy favors independent progress within the selected batch, not
strict processing order: a later message can complete before an earlier failure.
Retries are bounded per cycle and by their minimum post-completion interval, not
by a lifetime attempt count. There is no automatic attempt cap, dead-letter queue,
discard, or exponential backoff. Persistent failures remain pending and can fill
every batch slot, delaying messages beyond that selection. Applications own
monitoring, repair/removal, handler timeouts, and idempotency of external effects.

One poll loop runs per registered subscriber as before. Returned rows, handler
calls, and successful-ID buffering remain bounded by `BatchSize`; retries create
no acknowledgement records until success. The existing retained-history scan
and storage costs from F5 remain unchanged. Metadata decoding retains its existing
empty-map fallback. Returning nil is the application's success signal, not proof
that an external side effect occurred exactly once.

A crash or persistence failure after handler success can still repeat delivery.
Previously acknowledged failures from older versions are not automatically
recovered. The [Pubsub reference](../../../docs/reference/pubsub/README.md#failures-and-retries),
how-to, User Guide, and `Unreleased` describe the delivered policy and its limits.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`. Database tests used an isolated native
PostgreSQL 18.6 cluster on loopback, stopped after validation. These are local
results, not a claim that the PostgreSQL 16 CI job ran.

- Before the fix, `go test ./pubsub/postgres -run '^TestRetryBatch$' -count=1`
  failed all six named cases: failures in the first, middle, and last positions
  were acknowledged, both with and without the planned restart.
- `go test ./pubsub/... -count=1 -timeout=60s`: passed.
- `go test -race ./pubsub/... -count=1 -timeout=60s`: passed.
- `go test -race ./pubsub/postgres -run '^(TestRetryBatch|TestRetryBounds|TestRetryFanout|TestInvalidPayload|TestRetryLoop)$' -count=50 -timeout=60s`:
  passed all 50 repetitions, including real polling cadence and cancellation.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.2%; `pubsub/postgres`: 91.5%.
- `make docs-check`: passed.
- `git diff --check`: passed.

Deterministic direct-poll tests cover failure position, transient success,
durable acknowledgement contents, diagnostic offsets, named restarts, persistent
failures at batch sizes 1/2/3, independent healthy subscribers, and malformed
payload retention followed by repair. A channel-coordinated real poll-loop test
checks that a slow failure still waits the configured interval before retry,
then verifies cancellation and absence of acknowledgement. F5's late-commit,
migration, fan-out, and atomic rollback regressions continue to pass.

The previous sleep-based handler-error test asserting two non-retried attempts
was removed; these tests assert the corrected behavior at persistence and polling
boundaries instead.

## Boundary

Only finding F6 is addressed. There is no panic recovery, retry scheduler,
dead-letter store, retention manager, cross-process subscriber claim, or
exactly-once guarantee. Unrelated architecture-review findings are unchanged.

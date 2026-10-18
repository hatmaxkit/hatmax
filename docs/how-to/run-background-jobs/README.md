<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Run Background Jobs

Use `scheduler.Runner` when work must be polled from a durable `JobStore` and
executed outside an HTTP request.

## Create the runner

With a started database provider exposing `GetDB() *sql.DB`, root
`*config.Config`, logger and optional `scheduler.SettingsProvider`, create the
store and runner. Apply `schedulerpostgres.Schema` in a prior migrator before
polling:

```go
store := schedulerpostgres.NewStore(database.GetDB())
runner := scheduler.NewWithConfig(store, settingsService, cfg, logger)
```

Register one handler per task type:

```go
runner.Register("send-report", func(ctx context.Context, job scheduler.Job) scheduler.Result {
	return scheduler.Result{Output: map[string]any{"sent": true}}
})
```

This handler demonstrates recorded output. Replace it with the application
workflow before treating `sent` as a report-delivery claim.

Place the runner after the database and any component that installs the
scheduler schema. The runner implements `Startable` and `Stoppable`.

## Enable polling

Set:

```yaml
scheduler:
  enabled: true
  interval: 30s
  batch_size: 20
  workers: 2
  retry_attempts: 3
  retry_backoff: 1m
```

## Define the schedule

For one-shot work, leave `schedule_spec` empty. For recurring work, insert a
rule with the initial due time:

```sql
INSERT INTO scheduled_jobs (id, name, task_type, next_run_at, schedule_spec)
VALUES ('daily-report', 'daily-report', 'send-report', NOW(),
        '{"type":"daily","hour":9,"minute":0}');
```

Daily and weekly rules use `schedule_tz`, defaulting to UTC. Interval rules use
a positive Go duration, such as `{"type":"interval","every":"30m"}`.
In fake-store tests, set `Job.Schedule` to the corresponding `scheduler.Daily`,
`scheduler.Weekly`, or `scheduler.Interval`; leave it nil for one-shot work.

The store records the terminal result and advances or retires the slot
atomically. Recurring work advances from completion time after success or
exhausted failure; missed occurrences are skipped. Do not manually advance the
schedule from a handler. Retry attempts bound total attempts per slot, including
the first; backoff delays the next eligible poll after a failure. Retry waits
are persisted without advancing the schedule or holding a worker.

## Verify the result

Insert a due job with a registered `task_type`, start the process, and inspect
its `job_runs` row. It should move to `success` with JSON output, `retry_wait`
while a failed handler has attempts left, or `failed` after its budget is
exhausted. Trigger a handler panic and verify retry or failure with
`handler panic:` detail.
A later healthy job should still execute. Check logs for failed-state write
errors before assuming that the failure was persisted.

Fail once and then succeed: verify a stable run ID, incrementing `attempt`, and
no attempt before `retry_at`. Restart with a different configured retry limit
and confirm that the stored slot budget remains unchanged. Set
`retry_attempts: 1` when a single failure should be terminal.

Tick again at the same time and confirm the old slot does not execute again.
For one-shot work, check `enabled = false`; for recurring work, check the new
`next_run_at` and run at that time to confirm a second distinct slot. In legacy mode, claimed
but unfinished runs stay excluded from due batches and require explicit repair.

## Enable recovery for new attempts

After applying the additive schema, opt into durable execution:

```go
policy := scheduler.DefaultExecutionPolicy()
runner := scheduler.NewWithConfig(store, settingsService, cfg, logger,
    scheduler.WithDurableExecution(policy))
```

Register handlers before startup. `Start` and direct `Tick` validate the policy and
store capability. The first claim saves the job's policy; change future slots with
`store.ConfigureDurable(ctx, jobID, policy)`. Existing runs retain their snapshots.

Stop one worker after it claims a due slot, then start another worker. After the
stored lease expires, confirm the same run ID and slot, a new attempt and token,
and a successful or retry-waiting result. Never shorten a production lease merely
to bypass an active handler. A deadline is cooperative; protect repeated database
effects transactionally and external effects with application idempotency keys.

## Upgrade and adopt legacy runs

1. Stop every old scheduler worker and confirm all handlers have returned. Keep
   their database open until shutdown completes. Mixed old/new ownership is unsupported.
2. Apply `schedulerpostgres.Schema` or the equivalent additive migration. Existing
   rows remain legacy, including their attempt counts and raw diagnostics.
3. Inspect `store.ListRuns(ctx, schedulerpostgres.RunQuery{JobID: jobID})`. Check
   the run's state, `Attempt` and `UpdatedAt`, its current job slot, and whether
   its application effects already happened. Resolve domain effects before retrying.
4. For an unfinished current slot, choose a valid policy with `MaxAttempts` greater
   than its consumed attempt count. Call
   `store.AdoptLegacyRun(ctx, run.ID, run.Attempt, run.UpdatedAt, policy)` using the
   inspected revision. Terminal, stale, disabled or superseded work is refused.
5. Start only the upgraded durable workers. Adoption consumes no extra attempt
   itself, preserves a future retry wait, and permits the next claim to increment
   the retained count. Verify the retained code and subsequent terminal outcome.

No automatic scan declares legacy workers abandoned or invents historical budgets.
If the current attempt already consumed the proposed budget, resolve that run
explicitly before selecting more attempts; adoption will refuse it.

## Inspect and retain history

Use `ListRuns` with a limit up to 100 and the previous row's `CreatedAt`/`ID` cursor
for the next page. `expired` exposes reclaimable ownership without exposing tokens,
payloads, output or arbitrary error text. Filter by job/state to investigate work.

Run bounded terminal cleanup with an application-selected cutoff:

```go
pruneCtx, cancelPrune := context.WithTimeout(ctx, 5*time.Second)
defer cancelPrune()

removed, err := store.PruneRuns(pruneCtx, time.Now().Add(-30*24*time.Hour), 1000)
if err != nil {
    return err
}
logger.Infof("pruned %d terminal runs", removed)
```

Repeat batches when needed. Pending, running and retry-waiting runs remain;
retention does not run in a hidden background service.

## Shut down polling

Use a fresh timeout context for shutdown, not the canceled application context:

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := runner.Stop(shutdownCtx)
if err != nil {
	logger.Errorf("scheduler shutdown: %v", err)
}
```

`Stop` cancels active execution and waits for the admitted batch. Handlers must
observe `ctx.Done()` and pass that context to blocking operations. A timeout
bounds the wait, not the handler's lifetime; keep the store alive and wait again
before assuming shutdown is complete. Inspect interrupted run claims before
explicit repair or rescheduling. Repeated stops are safe. Construct a new
runner rather than calling `Start` on one that has begun shutdown.

See [Scheduler Reference](../../reference/scheduler/README.md).

## Coordinate retained background work

Keep scheduled-run cleanup and event-message cleanup explicit and separate. A
terminal scheduled run does not prove that every subscriber processed an event,
and a successful transport acknowledgement is not a domain-effect receipt.
Inspect classified outcomes and reconcile the application's effects before replay
or legacy adoption. Use finite contexts and bounded cleanup batches for both.
The [event operating procedure](../use-pubsub/README.md#inspect-and-replay-retained-failures)
covers revision-checked replay, subscriber retirement and fan-out-aware retention.

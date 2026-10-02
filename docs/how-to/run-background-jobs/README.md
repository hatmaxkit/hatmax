<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Run Background Jobs

Use `scheduler.Runner` when work must be polled from a durable `JobStore` and
executed outside an HTTP request.

## Create the runner

With an existing `*sql.DB`:

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
failure; missed occurrences are skipped. Do not manually advance the schedule
from a handler. Configured retry fields are not applied.

## Verify the result

Insert a due job with a registered `task_type`, start the process, and inspect
its `job_runs` row. It should move to `success` with JSON output or `failed`
with the handler error.
Trigger a handler panic and verify a failed run with `handler panic:` detail.
A later healthy job should still execute. Check logs for failed-state write
errors before assuming that the failure was persisted.

Tick again at the same time and confirm the old slot does not execute again.
For one-shot work, check `enabled = false`; for recurring work, check the new
`next_run_at` and run at that time to confirm a second distinct slot. Claimed
but unfinished runs stay excluded from due batches and require explicit repair.

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

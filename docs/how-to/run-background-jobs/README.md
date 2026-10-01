<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
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

The current runner records each attempt but does not calculate or persist the
next run and does not apply retry configuration. The application or store must
schedule later runs explicitly.

## Verify the result

Insert a due job with a registered `task_type`, start the process, and inspect
its `job_runs` row. It should move to `success` with JSON output or `failed`
with the handler error.
Trigger a handler panic and verify a failed run with `handler panic:` detail.
A later healthy job should still execute. Check logs for failed-state write
errors before assuming that the failure was persisted.

See [Scheduler Reference](../../reference/scheduler/README.md).

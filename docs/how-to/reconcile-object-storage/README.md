<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Reconcile a Failed Object Mutation

Use this procedure when `Put` or `Delete` fails and the application must decide
whether to retry or complete a durable workflow. Keep the operation's target and
intended content identity in application state; never use a grant as a durable
record. Inspect errors with `errors.Is`, preserving the original cause.

| Predicate or cause | Meaning and next action |
| --- | --- |
| `storage.ErrInvalid` | Reject the request or fix configuration before I/O. |
| `storage.ErrTooLarge` | Reject the admitted size; do not accept a truncated read as complete. |
| `storage.ErrNotFound` | An Open/Stat observed confirmed absence; this is not permission denial. |
| `storage.ErrUnsupported` | Choose a supported configured capability; do not weaken the policy. |
| `storage.ErrIndeterminate` | The mutation may have committed. Reconcile before retrying. |
| `context.Canceled` / `context.DeadlineExceeded` | Stop the current operation; if also indeterminate, its remote result remains unknown. |
| Authentication, permission, malformed metadata or transport error | Retain the cause; never turn it into confirmed absence. |

1. Stop automatic mutation retries. The S3 adapter makes one mutation attempt.
   `ReadAttempts` applies only to read-only operations; the same operation context
   and timeout bound every attempt and returned body.
2. For an indeterminate write, read the exact intended key with a fresh bounded
   context. Read to successful EOF, close the reader and validate actual bytes
   against the application's intended identity. An ETag or content type is not
   a content hash. A successful `Stat` alone does not prove intended contents.
3. For an indeterminate delete, use Open/Stat to confirm absence. Permission or
   timeout still leaves the outcome unknown. Delete is idempotent for confirmed
   absence and does not remove historical versions.
4. Decide the retry or compensation in application state. Use distinct immutable
   destinations if overwriting concurrent work would be unsafe. Local atomic
   replacement and remote writes are not compare-and-swap or exactly-once.
5. Recheck current authority before publishing availability. A completed store
   mutation does not update quotas, ownership, records or application completion.

On read failures close the owned reader; on Put failures the input reader still
belongs to the caller. Spool cleanup is adapter-owned. Context checks cannot
interrupt an arbitrary already-blocked caller `io.Reader.Read`; give such readers
appropriate I/O cancellation or deadlines rather than detached goroutines.

See [Storage Reference](../../reference/storage/README.md) and
[Snapshot and Bearer Ownership](../../explanation/private-object-ownership/README.md).

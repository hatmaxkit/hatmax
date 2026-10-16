<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Bounded Local Objects

Status: reviewing
Delivery set: object-storage
Plan: [Object storage plan](../../../plan/object-storage.md)
Tracker: [Object storage tracker](../../../tracker/object-storage.md)
Branch: `feat/bounded-local-storage`
PR: `#123`

## Purpose

Deliver bounded private document storage through a small neutral contract and
a working filesystem adapter, with real confinement and resource evidence.

## Delivered Behavior

- Write, inspect, open and delete empty or non-empty objects without image
  dimensions or public URLs. Keys are validated literally before object I/O.
- Reject invalid bounds and oversized input; preserve previous complete bytes
  when input fails. Complete temporary writes replace destinations atomically.
- Return metadata from the opened file and an independently owned reader.
  Enforce actual streamed bounds even if a file grows after it is opened.
- Keep reads, writes, inspection and deletion confined under symlink replacement.
  Confirmed absence, permission errors and uncertain mutations remain distinct.

## Implementation Notes

T1.1 is `b6ba9e4509918ec2fe6fca7a491a63a552ee10fd`; T1.2 is
`94681613bb9ef8075979dd9b0ede1d99500b8259`.
`os.Root` owns traversal. Each write pins its parent directory through creation,
rename and cleanup. Random exclusive spool files use mode 0600; new directories
use mode 0700. A write's explicit maximum cannot exceed configured
`MaxSpoolBytes`; N concurrent writes can occupy N times that capacity, plus
filesystem overhead. Copying uses fixed-size buffers, not maximum-sized memory.

Local content types are derived with `http.DetectContentType` from at most
512 bytes, including after reopening. A valid declared content type is a hint
and is not persisted locally. Neither metadata nor a filename validates content.
The application controls the root, initial ancestors, hard links and mounts.
Atomic replacement is neither compare-and-swap nor crash-durable publication.

## Contracts Added or Changed

`storage` adds `Object`, `WriteOptions`, `ReadOptions`, `Store`, canonical key
validation and the five specified error predicates. `storage/local.NewStore`
requires an existing root and finite spool capacity; `Close` releases its root
handle without taking ownership of already opened readers. `Put` never closes
the caller's input. Cancellation is checked around synchronous reads and phases;
it cannot interrupt an arbitrary already-blocked caller reader. No detached
I/O goroutine emulates cancellation. Mutations are not automatically retried.

## Files of Interest

- `storage/store.go`: neutral values, option validation, keys and error contract.
- `storage/local/store.go`, `storage/local/write.go`: confined I/O and replacement.
- `storage/local/*_test.go`: real lifecycle, link-race, byte and ownership checks.
- `storage/internal/stream`: actual byte enforcement and cancellation boundaries.
- `scripts/check-object-storage.sh`: delivered Slice 1 checks and retained receipts.

## Validation

Focused T1.2 check passed:
`GOWORK=off go test -count=1 -timeout=60s ./storage/...`.

The final PR uses these ordered controller-recorded commands:

- `GOWORK=off scripts/check-object-storage.sh slice 1`
- `GOWORK=off make docs-check`
- `git diff --check`

The Slice 1 runner passed through the controller at
`07af2e04d77834fb104adb7f6a02041c93b471dc`. Documentation validation then found a
stale tracker link after the source-ticket move; the link was corrected before
regenerating final evidence. The controller's exact clean-HEAD receipt owns the
final result for all three commands. The slice
runner retains command/log hashes and explicit passes for TestKeys, TestBounds,
TestObjects, TestConfinement and TestWriteFailure in ordinary and race runs,
plus existing image/local regressions, licensing, formatting, shell syntax,
affected build, vet and configured lint. Missing or skipped roots fail.
Full mode is rejected until its owning task delivers complete acceptance.
The runner records initdb, pg_ctl and psql availability as later full-gate
prerequisites; Slice 1 does not start a database or remote fixture.

## Risks and Follow-ups

Filesystem writes provide atomic visibility, not power-loss durability. Callers
own concurrency admission and aggregate disk quotas. Remote storage, signing,
executable guides and complete acceptance retain their approved later owners.

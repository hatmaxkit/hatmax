<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Local Image Root Confinement

Status: reviewing
Ticket: [TKT-20260930211801](../ticket/reviewing/20260930211801-confine-local-image-storage-to-its-root.md)
Branch: `fix/ticket-20260930211801-local-image-root`
PR: pending

## Delivered Behavior

- `Put`, `Get`, and `Delete` operate through `os.Root` handles rather than
  joining unchecked keys to the configured root.
- Empty, absolute, lexically escaping, and root-equivalent keys return an `os.PathError`
  wrapping `fs.ErrInvalid` before the root is created or touched.
- Accepted keys are normalized; nested paths and contained relative symlinks
  remain supported. Escaping and absolute symlinks cannot redirect reads or
  writes outside the opened root.
- Parent creation is also confined. Replacing a path component with an
  escaping symlink cannot create directories or files outside the root.
- Deleting a final symlink unlinks it without following or deleting its target.

## Contracts and Ownership

The constructor, `image.Store` interface, `URL`, and `BasePath` are unchanged.
Construction performs no filesystem work. `Put` still creates a missing root
and parents; reads and deletes do not create the root. Each call closes its
root handle. A reader returned by `Get` owns an independent file handle that
the caller must close.

The configured root and its ancestors are trusted application configuration;
they must not be attacker-controlled. Per-call roots do not pin one directory
identity across successive operations. Filesystem sandboxing against hard
links, privileged mounts, or device files is not introduced. Symlink-race
protection inherits `os.Root`'s platform guarantees; `GOOS=js` does not provide
that protection.

The [local store reference](../../../docs/reference/image/README.md#local-store)
documents these rules. The implementation uses the standard library's
[traversal-resistant APIs](https://go.dev/blog/osroot), without new dependencies.

## Validation

All Go commands used `GOTOOLCHAIN=go1.26.7`; filesystem tests ran on Linux.

- Before the fix, `go test ./image/local -run '^TestStoreEscape$' -count=1`
  failed in all six traversal and directory-symlink cases. Reads succeeded,
  writes changed outside sentinels, and deletes removed them.
- `make check`: passed, including source licensing, formatting, vet, all tests,
  coverage, and strict lint. Total coverage: 80.1%; `image/local`: 100%.
  Database tests used an isolated native PostgreSQL cluster on loopback,
  stopped after validation.
- `make docs-check`: passed.
- `go test -race ./image/... -count=1`: passed.
- `go test -race ./image/local -run '^TestStoreLinkSwap$' -count=20`: passed.
  Atomic link replacements did not expose or modify outside sentinels, or
  create outside parent directories.
- `go test ./image/local -run '^$' -fuzz '^FuzzStoreKeys$' -fuzztime=10s -parallel=2`:
  passed, 9,023 executions. Outside sentinels and directory contents remained
  unchanged for arbitrary object keys.
- `make lint-strict`: passed.
- `git diff --check`: passed.

## Boundary

Only finding F2 is addressed. Image ingestion limits, S3 behavior, processing,
URL construction, and the other architecture-review tickets are unchanged.

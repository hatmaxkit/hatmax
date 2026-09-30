---
id: TKT-20260930211801
title: Confine local image storage to its root
status: reviewing
kind: bug
severity: high
priority: high
scope: persistence
tags: architecture-review, persistence, hardening
source: review
reported_at: 2026-09-30T21:18:01Z
ready_at: 2026-09-30T22:41:15Z
started_at: 2026-09-30T22:41:15Z
reviewed_at: 2026-09-30T22:51:18Z
branch: fix/ticket-20260930211801-local-image-root
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/69
commits: 572354eed81bef370fb9dab3f8847915eb4437fe
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Put, Get, and Delete join an unchecked key to basePath. A ../sentinel key accesses the parent directory. A symlink beneath the configured root can also redirect Put outside that root.

Sources: `image/local/store.go:31-57`.

Evidence: TestObservedStorageEscape reproduced traversal and symlink escape inside owned temporary directories.

Impact: Applications that accept user-derived storage keys can expose filesystem reads, overwrites, and deletion with the process permissions.

## Expected Outcome

Enforce root-confined operations for all three methods, including symlink traversal. Reject invalid keys and preserve valid nested object paths.

## Validation

Test traversal, nested keys, symlinks, and attempts to replace path components. Assert that outside-root sentinels remain unchanged.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f2).

## Implementation

All filesystem operations use per-call `os.Root` handles. Object keys must be
local and must not normalize to the root itself. Valid nested keys and relative
contained symlinks remain supported. Root creation stays lazy and the public
constructor is unchanged. External final symlinks can be unlinked without
following their targets.

Delivery: [Local image root confinement](../../report/20260930225118-local-image-root-confinement.md).

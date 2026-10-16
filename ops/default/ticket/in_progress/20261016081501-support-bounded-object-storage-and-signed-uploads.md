---
id: TKT-20261016081501
title: Support bounded object storage and exact-target signed uploads
status: in_progress
kind: task
severity: medium
priority: normal
scope: api
tags: storage, s3, signing
source: chat
reported_at: 2026-10-16T08:15:01Z
ready_at: 2026-10-16T09:11:36Z
started_at: 2026-10-16T09:11:36Z
branch: feat/signed-storage-uploads
pr: "#123, #124"
commits: b6ba9e4509918ec2fe6fca7a491a63a552ee10fd, 94681613bb9ef8075979dd9b0ede1d99500b8259, 9de7c8aef2d8a487cf192ad3b5899e64157d2fb5, b85c9496d412f0a759e4640264ffa0689c2fdd0c, 2b0802dc245271aadab6b41ce6d6968dcb004db8, 25e7d3ca3f7a78bf0a5b765043e0ae1a148cc38a
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

At `82defb7b51f71436ae71cbb5b2e2c980bca359b1`, `image.Store` exposes basic
Put/Get/Delete/URL operations. Its local and S3 adapters can store bytes, but
there is no neutral object-inspection contract or narrowly constrained upload
signing capability. General document handling should not depend on image
dimensions or an image persistence record. URL construction does not establish
application authorization or enforce upload admission.

The already adopted AWS SDK exposes presigned POST policies with typed adapter
configuration for exact targets, expiry and content-length conditions. Those
capabilities are not currently expressed in Hatmax's public storage contracts.

## Expected Outcome

The maintainer approved [Object storage capabilities](../../spec/object-storage.md)
on 2026-10-16. Its independent [plan](../../plan/object-storage.md) and
[tracker](../../tracker/object-storage.md) govern delivery. Provide bounded private object
I/O, metadata inspection and exact-target signed POST uploads through small Go
contracts and local/S3 adapters. Preserve existing image APIs and leave domain
ownership, authorization, quotas and completion with applications.

The specification defines OS01-OS10, provider capability boundaries, compatibility
and real fixture evidence. Scope, five-slice controller execution and controller-owned integration were
authorized on 2026-10-16. Slices 1-2 are delivered and Slice 3 is reviewing. This source ticket tracks the formal
delivery until its exact gate passes and its implemented resolution is recorded.

## Validation

Use real filesystem operations and an isolated S3-compatible server to prove
byte limits, exact-key/type/size/expiry policy enforcement, error classification,
snapshot use, cancellation and owned cleanup. Preserve image adapter and processor
regressions, execute documentation examples and run the exact-candidate complete
gate recorded by the future approved plan. A fake policy evaluator or compile-only
check cannot establish actual signed remote upload behavior.

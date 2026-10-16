---
id: TKT-20261016081501
title: Support bounded object storage and exact-target signed uploads
status: solved
kind: task
severity: medium
priority: normal
scope: api
tags: storage, s3, signing
source: chat
reported_at: 2026-10-16T08:15:01Z
ready_at: 2026-10-16T09:11:36Z
started_at: 2026-10-16T09:11:36Z
reviewed_at: 2026-10-16T10:28:36Z
closed_at: 2026-10-16T10:42:47Z
resolution: fixed
validated_commit: c03a4f042507eb8e255571aa2781fcf7a42f0036
branch: test/object-storage-acceptance
pr: "#123, #124, #125, #126, #127"
commits: bd1bd280a1ae2c7409d748ed0c6953fc795ea1fa, 9cde0688161bab2a69ef6b9a49cb62054bef5ae6, 7161cc892c5f0e56ddd5eb11c9c96d647c8d43ee, 270d1a89d8d562dd84f8ec008d014c35fb1592d9, b6ba9e4509918ec2fe6fca7a491a63a552ee10fd, 94681613bb9ef8075979dd9b0ede1d99500b8259, 9de7c8aef2d8a487cf192ad3b5899e64157d2fb5, b85c9496d412f0a759e4640264ffa0689c2fdd0c, 2b0802dc245271aadab6b41ce6d6968dcb004db8, 25e7d3ca3f7a78bf0a5b765043e0ae1a148cc38a
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
authorized on 2026-10-16. All five slices are delivered through PRs #123-#127. The controller full gate
passed at `c03a4f042507eb8e255571aa2781fcf7a42f0036`; OS01-OS10 are accepted
and this source ticket is resolved with implemented behavior.

## Validation

`GOWORK=off scripts/check-object-storage.sh` passed through controller validation
`ORAZRFW4UDUF4TSQEPDZ47HPFF` at the validated commit above. Its 21 ordered
steps bind 81 concern roots and 1,175 repository roots, actual filesystem and
signed remote boundaries, image regressions, executable guides, complete make
check with 80.9% coverage, and confirmed MinIO/PostgreSQL cleanup. The two
explicit unrelated skips do not count as storage acceptance. See the canonical
[acceptance report](../../report/object-storage-acceptance.md) for retained
candidate, command, manifest, result and log identities.

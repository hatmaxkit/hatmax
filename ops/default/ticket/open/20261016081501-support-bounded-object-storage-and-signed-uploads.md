---
id: TKT-20261016081501
title: Support bounded object storage and exact-target signed uploads
status: open
kind: task
severity: medium
priority: normal
scope: api
tags: storage, s3, signing
source: chat
reported_at: 2026-10-16T08:15:01Z
commits:
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

Refine and approve [Object storage capabilities](../../spec/object-storage.md),
then deliver its independent plan and tracker. Provide bounded private object
I/O, metadata inspection and exact-target signed POST uploads through small Go
contracts and local/S3 adapters. Preserve existing image APIs and leave domain
ownership, authorization, quotas and completion with applications.

The specification defines OS01-OS10, provider capability boundaries, compatibility
and real fixture evidence. Capture does not approve its proposed package design
or start implementation. Keep this ticket open until formal scope approval;
record its eventual delivery artifacts and resolution through this repository's
normal lifecycle.

## Validation

Use real filesystem operations and an isolated S3-compatible server to prove
byte limits, exact-key/type/size/expiry policy enforcement, error classification,
snapshot use, cancellation and owned cleanup. Preserve image adapter and processor
regressions, execute documentation examples and run the exact-candidate complete
gate recorded by the future approved plan. A fake policy evaluator or compile-only
check cannot establish actual signed remote upload behavior.

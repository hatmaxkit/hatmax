---
id: TKT-20260930211819
title: Allow owned and bounded HTTP serving
status: open
kind: follow_up
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:19Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Serve creates a private http.Server with no header timeout or idle timeout and does not expose that server to callers. Shutdown accepts a different caller-owned server, so it cannot shut down the server created by Serve.

Sources: `app/lifecycle.go:91-104`.

Evidence: Source-confirmed. The canonical generated application is a non-affected comparison, not a reproduction of this helper.

Impact: The convenience serving path cannot share the normal lifecycle's server ownership or enforce basic connection bounds. Generated applications already use their own server and do not depend on this helper.

## Expected Outcome

Provide a serving path with caller-owned server configuration and shutdown identity, or explicitly deprecate the disconnected helper. Set or require an appropriate header timeout without breaking streaming behavior.

## Validation

Verify bounded incomplete headers, graceful shutdown of the exact serving instance, normal requests, and supported streaming behavior.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f20).

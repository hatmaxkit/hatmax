---
id: TKT-20260930211819
title: Allow owned and bounded HTTP serving
status: solved
resolution: fixed
closed_at: 2026-10-01T15:29:33Z
kind: follow_up
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:19Z
ready_at: 2026-10-01T14:46:15Z
started_at: 2026-10-01T14:46:15Z
reviewed_at: 2026-10-01T14:57:43Z
branch: fix/ticket-20260930211819-owned-http-server
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/87
commits: abfa220d93f6ea6e75a4cb5521a6654873bb6621, fe57453d71830be11f3ebca36dc21f10258d625b
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
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

## Selected Contract

Serve accepts a caller-owned *http.Server instead of a router and address, preserving the exact instance for Shutdown. Before listening, zero ReadHeaderTimeout and IdleTimeout values receive defaults of 5 and 60 seconds; positive caller values are preserved and negative values are rejected. ReadTimeout, WriteTimeout, handlers, hooks, address, and other server configuration remain caller-owned. No whole-response write deadline is imposed, so streaming stays supported. Repository examples and guidance migrate to the explicit server argument; generated applications already own their server and remain unchanged.

## Delivery

The serving helper now accepts the application-owned server, validates connection bounds before defaulting or listening, and preserves other server policy. Repository callers and documentation use the new signature. The Ticked example and bootstrap how-to pass the same server to Shutdown, allowing active HTTP requests to finish before dependencies stop.

`make check`, `make docs-check`, `go test -race ./app -count=1`, and `go test ./app -run '^TestServe' -count=20` passed. Total coverage is 81.6%; app coverage is 93.9%. Real TCP tests verify incomplete-header and idle connection limits, normal requests, streaming beyond connection deadlines, exact serving-instance shutdown, request draining, and dependency-stop ordering.

Report: [Owned HTTP Serving](../../report/20261001145743-owned-http-serving.md).

PR #87 was verified merged into dev at fe57453d71830be11f3ebca36dc21f10258d625b. F20 is resolved.

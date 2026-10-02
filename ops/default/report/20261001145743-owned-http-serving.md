<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Owned HTTP Serving

Status: delivered
Ticket: [TKT-20260930211819](../ticket/solved/20260930211819-allow-owned-and-bounded-http-serving.md)
Branch: `fix/ticket-20260930211819-owned-http-server`
PR: [#87](https://forge.adrianpk.com/hatmax/hatmax/pulls/87)
Implementation: `abfa220d93f6ea6e75a4cb5521a6654873bb6621`
Integrated into dev: `fe57453d71830be11f3ebca36dc21f10258d625b`

## Delivered Behavior

- Serve receives a caller-owned *http.Server rather than creating an inaccessible private instance. Applications retain that exact pointer for Shutdown, making serving and graceful shutdown share server identity.
- Zero ReadHeaderTimeout and IdleTimeout values default to 5 and 60 seconds before listening. Positive custom bounds are preserved; a nil server or negative connection limit returns an error before listening or partially applying defaults.
- ReadTimeout, WriteTimeout, handlers, address, hooks, and other server settings remain caller-owned. Streaming responses are not capped by a new whole-response timeout. Normal server closure remains a nil Serve result; other listen errors are returned unchanged.
- The Guide and Ticked examples, root README, bootstrap how-to, application reference, package guidance, User Guide chapter, and Unreleased use or explain the new signature. Ticked and the bootstrap how-to pass the same serving instance to Shutdown and wait for it before exit.

## Contracts and Ownership

The application owns server construction, configuration, serving invocation, and shutdown. Configure the server before calling Serve and do not mutate it while serving. Migrate Serve(router, port) to a retained &http.Server{Addr: port, Handler: router} passed to Serve(server). No compatibility wrapper or second hidden server is introduced.

Header and idle deadlines bound incomplete headers and idle keep-alive connections, not active handlers or response streams. Applications still own request-body bounds, per-handler cancellation, response timeout policy, and upgraded/hijacked connections. Serve can return when the listener closes while Shutdown is still draining requests; a Serve return alone is not completion of graceful shutdown.

The existing Shutdown timeout, stop ordering, and error policy remain unchanged. Ticked now uses that shared shutdown boundary instead of stopping components while its private serving listener remained open. Generated applications already construct their server and use the standard library directly; their rendering and runtime contracts are unchanged.

Diataxis keeps exact signature, limits, migration, and lifecycle semantics in the reference. The existing bootstrap procedure and User Guide chapter explain ownership in their current locations without restructuring documentation.

## Validation

All checks used Go 1.26.7. Full repository database tests used an isolated native PostgreSQL 18.6 cluster, stopped after validation. HTTP behavior tests used real ephemeral loopback TCP listeners; BaseContext exposed listener readiness without reserving/releasing a port or polling startup.

- `make check`: passed source licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.6%; app: 93.9%.
- `make docs-check`: passed.
- `go test -race ./app -count=1`: passed.
- `go test ./app -run '^TestServe' -count=20`: passed all 20 repetitions.
- `git diff --check`: passed.

Named cases verify default and custom connection policy, independent defaulting, unchanged caller configuration, invalid-policy rejection without partial mutation, normal closed-server exit, preserved underlying listen failures, incomplete-header closure before handler invocation, successful requests followed by idle closure, SSE output beyond header/idle deadlines, shutdown of the exact serving listener, refusal of new connections, active request draining, and dependency-stop ordering. Existing component startup, rollback, and shutdown tests continue to pass.

## Boundary

Only F20 is corrected. There is no new TLS serving wrapper, listener abstraction, whole-request deadline, forced-close policy, dependency, generator change, or change to the remaining findings. Request-body limits and stream-specific lifecycle remain application responsibilities.

<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Published Scaffold Baseline

Status: delivered
Ticket: [TKT-20260930211821](../ticket/solved/20260930211821-refresh-the-published-scaffold-dependency-baseline.md)
Review: [F22](20260930211800-architecture-nit-review.md#f22)
Branch: `fix/ticket-20260930211821-scaffold-baseline`
PR: [#90](https://forge.adrianpk.com/hatmax/hatmax/pulls/90)
Implementation: `ac9355a08a966aa129f84cbcc481560b685f08bf`
Integrated dev: `352bd5a7f97541f1129bd172e057a03cf8feb998`

## Delivered Behavior

- New applications explicitly require pgx v5.11.0 and x/text v0.42.0, with their published archive and module checksums. The supported published Hatmax version remains v0.5.0; chi remains v5.3.2 and the Go baseline remains 1.26.
- Module validation resolves the complete graph through the existing go mod tidy step. Standalone consumer acceptance checks the selected published versions, rejects all replacements, and reruns generated tests in readonly module mode.
- `make generator-scaffold-security` scans a real generated consumer with govulncheck, rather than treating the checkout's dependency graph as proof about generated applications.
- The existing generator reference documents the baseline and validation commands. Unreleased describes the corrected dependencies and the separate update requirement for existing applications.

## Contract and Ownership

The application renderer owns explicit requirements and checksum seeds. Go module version selection overrides the older transitive versions selected by published Hatmax v0.5.0. No Book version, release range, module replacement, new Hatmax publication, or dependency in the Hatmax checkout changes.

The original generated consumer scan reached symbols affected by [GO-2026-5004](https://pkg.go.dev/vuln/GO-2026-5004) and [GO-2026-5970](https://pkg.go.dev/vuln/GO-2026-5970). These advisories are fixed before the selected pgx and x/text versions. Scanner reachability is not proof of a concrete exploit.

## Validation

Checks used Go 1.26.7. Database tests used an isolated native PostgreSQL 18.6 cluster, stopped after validation. Published scaffold and real application creation used Hatmax v0.5.0 with no local replacement. The existing feature acceptance uses the local Hatmax checkout and is not the published-consumer security proof.

- Before the fix, the baseline regression reproduced missing explicit requirements and checksum seeds. The published-consumer scan reported two reachable vulnerabilities.
- `make check`: passed licensing, formatting, vet, all tests, the 80% coverage gate, and strict lint. Total coverage: 81.7%; generator/execute: 79.1%. The first invocation stopped in licensing because the moved ticket was not yet staged; staging the lifecycle move resolved that index-path check before the successful full run.
- `make generator-scaffold-acceptance generator-scaffold-security`: passed. The generated graph selects Hatmax v0.5.0, pgx v5.11.0, and x/text v0.42.0 with no replacements. govulncheck reports zero reachable vulnerabilities and zero affected imported packages, plus 17 advisories in required modules whose affected symbols are not called.
- `HATMAX_REAL_APPLICATION_COMMANDS=1 GO_TEST_COMMAND='go test ./...' go test -v -count=1 -run '^TestTerminalSurfaceCreatesTerseAndDetailedApplications$' ./internal/hatmaxcli`: passed both terse clarified and detailed composite creation. Real Go/sqlc commands, compilation, template startup, and PostgreSQL-backed generated tests ran.
- `make generator-project-acceptance`: passed real feature generation, formatting, project checks, and rejection of deliberately uncompilable output.
- `make docs-check`: passed local links, documentation structure, example compilation, and whitespace.
- `go test ./generator/execute ./generator/book ./generator/plan -count=1`: passed.
- `golangci-lint run --build-tags=acceptance --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./generator/execute/...`: passed with zero issues.
- `git diff --check`: passed.

Final integration validation repeated the aggregate, documentation, scaffold, security, real application creation, and existing-project feature checks on dev commit 352bd5a7f97541f1129bd172e057a03cf8feb998. `go test -race ./...` also passed. The checkout scan found zero reachable vulnerabilities and zero affected imported packages, plus one module-only advisory; the published scaffold scan still reports 17 module-only advisories. The owned database cluster was stopped after this validation. The [review closure](20260930211800-architecture-nit-review.md#delivery-closure) records the complete F1-F22 result.

## Boundary

The initial composite acceptance exposed missing template-function registration. That independently approved defect was fixed and integrated in [PR #89](https://forge.adrianpk.com/hatmax/hatmax/pulls/89) before repeating F22 acceptance; its implementation is not in this diff.

Existing applications are not rewritten automatically. Module-only advisories remain distinct from called vulnerable symbols. Updating the scaffold's transitive requirements does not publish the checkout's other fixes as a new Hatmax library release. Tags, main alignment, mirrors, and unrelated open tickets are outside this delivery.

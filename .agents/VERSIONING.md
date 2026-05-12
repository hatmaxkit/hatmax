# Versioning Rules

This document defines commit, branch, and merge rules.

## Commit format

Use Conventional Commits:
- `<type>(<optional-scope>): <description>`
- common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`

Examples:
- `feat(core): add project state persistence`
- `fix(cli): handle missing argument error`
- `chore(ops): close slice-2 tracker status`

## Remote policy

- `origin` is the canonical remote for the project workflow.
- Open and manage pull requests against the configured repository forge.

## Branch policy

- `main` is protected; no direct commits.
- `dev` is the mandatory trunk branch for all planned and day-to-day delivery work.
- Day-to-day work happens on `dev` or short-lived branches created from `dev`.
- Merge to `main` only through explicit `dev -> main` release-alignment pull requests.
- Slice branches should use: `fix/slice-<n>-<short-slug>`.

## Pull request base policy

- Default and mandatory PR target for slices, tickets, and planned implementation is `dev`.
- PRs targeting `main` are not allowed during normal delivery flow.
- `dev -> main` is a formal alignment protocol executed only on explicit user request.

## Alignment policy (`dev` and `main`)

- Regular implementation flow is linear through `dev`.
- `main` receives changes only from explicit alignment actions requested by the user.
- `dev -> main` alignment uses rebase + fast-forward.
- If an unintended PR targets `main`, stop and realign by replaying the missing commit chain into `dev` without destructive history edits.

## Workflow policy

- Prefer small, reviewable commits.
- Keep history linear where practical.
- Avoid history rewrites unless explicitly approved.
- Keep commit scope aligned with one active slice/task concern.
- `ops/plans/*` and `ops/trackers/*` maintenance commits are allowed directly on `dev` when they do not include runtime code changes.

## Git safety

- Never run destructive operations without explicit approval.
- Keep working tree clean before declaring a delivery cycle complete.

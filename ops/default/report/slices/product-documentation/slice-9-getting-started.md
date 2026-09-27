# Slice 9: Getting Started

Status: delivered
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-9-getting-started`
PR: `#12`

## Purpose

Teach the first running Hatmax process and list that chapter from the User
Guide.

## Delivered Behavior

A reader can open Getting Started from the User Guide index. The chapter
loads configuration, creates a logger and a router, and reaches `app.Start`.
The check is `GET /ping` returning `{"status":"ok"}` while the process is
still running.

## Implementation Notes

The sample does not pass components to `app.Setup`. `config.Validate` still
accepts the default database fields. The chapter says it does not connect to
Postgres.

## Contracts Added or Changed

The Getting Started chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/getting-started.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

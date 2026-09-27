# Slice 10: Postgres Chapter

Status: reviewing
Delivery set: product-documentation
Plan: `ops/default/plan/product-documentation.md`
Tracker: `ops/default/tracker/product-documentation.md`
Branch: `docs/slice-10-postgres-chapter`
PR: pending

## Purpose

Teach adding a Postgres component and show that a failed start exits before
the process listens.

## Delivered Behavior

A reader can open Add Postgres from the User Guide index. A closed database
port makes the process exit with `cannot ping database`. A reachable database
keeps the process up, and `GET /ping` still returns `{"status":"ok"}`.

## Implementation Notes

The sample passes only the database component to `app.Setup`, so a failed
start has no earlier component to stop. The chapter says that. It does not
run migrations.

## Contracts Added or Changed

The Add Postgres chapter. No Go contracts changed.

## Files of Interest

- `docs/tutorials/user-guide/postgres.md`
- `docs/tutorials/user-guide/index.md`

## Validation

- `git diff --check` passed for both task commits.
- Relative links in `docs/` resolve to files in the tree.

## Risks and Follow-ups

None recorded.

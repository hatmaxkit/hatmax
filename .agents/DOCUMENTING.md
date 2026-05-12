# Documentation Guidelines

This repository uses Diataxis for user-facing documentation under `docs/`:
https://diataxis.fr

## Categories

- `docs/tutorials/` for learning paths
- `docs/how-to/` for task procedures
- `docs/reference/` for contracts and exact behavior
- `docs/explanation/` for rationale and tradeoffs

## Examples and references

- Keep setup and execution examples aligned with the stack, package manager, runtime, and entrypoints actually present in the project.
- API and reference docs should use the naming, signatures, paths, and conventions used by the codebase.
- If API docs are generated, keep source comments compatible with the project's documentation tool.

## Operational documents

- `ops/specs/` stores formal specs and drafts.
- `ops/conv/quiz/` stores raw and curated quiz artifacts.
- `ops/conv/dump/` stores raw and curated knowledge dumps.
- `ops/plans/` stores cycle plans.
- `ops/trackers/` stores execution trackers.
- `ops/artifacts/` stores durable cycle outputs and reports.
- Do not mix operational delivery records into `docs/` unless they become durable product documentation.

## General rules

- Keep all repository docs in English.
- Each document should have one primary intent.
- Split documents when intent changes.
- Preserve technical meaning when reorganizing docs.
- Avoid title/heading suffixes in parentheses (for example `(Draft)`).
- Prefer semantic metadata fields (for example `Status`) over title decoration when state needs to be explicit.
- For structural documentation updates, complete one consistency pass before commit.
- Prefer one cohesive commit per documentation iteration; avoid corrective micro-commits caused by partial updates.

## Scope boundary

Diataxis applies to `docs/*` only.
Operational instructions in `.agents/*` and delivery records under `ops/*` follow their own templates.

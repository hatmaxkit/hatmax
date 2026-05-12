# Version Control Workflow

1. Create a branch from `dev`.
2. For slice execution, use `fix/slice-<n>-<slug>`.
3. Never work on `main` directly.
4. Before opening a PR from `dev` to `main`, run the configured release-alignment gate.
5. Merge into `main` only via release-alignment PR from `dev`, using rebase + fast-forward. This is mandatory for every `dev -> main` alignment, including docs-only changes.
6. Docs-only gate exception: when the `dev -> main` diff contains only Markdown repository documentation, skip runtime tests and run whitespace/diff validation instead.
7. If release metadata, badges, or generated status files become stale during the gate, commit refreshed outputs before PR.
8. Avoid force push unless explicitly approved.
9. Exception: `ops/plans/*` and `ops/trackers/*` updates may be committed directly to `dev` when they only formalize planning/tracking state.

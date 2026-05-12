# Slice-Based Development Strategy

This document defines delivery strategy for non-trivial work.
Use the Slice Delivery playbook for the step-by-step execution procedure.

## 1. Definitions

- Task: high-level requested work unit
- Slice: smallest meaningful, testable behavior increment
- Commit: atomic versioning unit

## 1.1 Naming Convention (Mandatory)

Use one naming scheme consistently across plans, trackers, tickets, PR notes, and reports.

- Slice labels: `Slice 1`, `Slice 2`, `Slice 3`, ...
- Task labels: `T1.1`, `T1.2`, `T2.1`, ...
- Completion reporting: `Slice N of M completed`

Do not mix alternative numbering styles for these entities:

- no alphabetic slice/task labels (`Slice A`, `Task B2`)
- no Roman numerals (`Slice IV`, `Task II.3`)

## 2. Decomposition rules

Use a single slice only when scope and integration risk are low.
Otherwise decompose into multiple behavior-first vertical slices.

## 3. Slice rules

Each slice must:
- deliver a functional increment
- be testable in isolation
- integrate without breaking existing behavior

## 4. Commit strategy

- Keep commits cohesive and scoped.
- Do not mix unrelated slice concerns in one commit.
- Keep commit subjects aligned with slice task intent.
- Use conventional commits.
- For approved slice execution, use one commit per tracker task.

## 5. Integration

- Integrate slices in safe order through PRs targeting `dev`.
- Keep repository passing checks after each integrated slice.
- Do not integrate slices directly into `main`.
- Align `dev -> main` only through the release-alignment playbook.

## 6. Traceable execution

For approved multi-slice work:
1. Create a dedicated branch: `fix/slice-<n>-<slug>`.
2. Implement only that slice's tasks.
3. Keep a one-to-one mapping between task IDs and commits.
4. Open one PR scoped to that slice.
5. Never open PRs per task or per commit.
6. Update `ops/trackers/*` with tasks, commits, and validation outcomes.
7. Complete review (manual and/or automated) before merge.
8. Merge the approved slice PR into `dev`.
9. Report slice status before starting the next slice.

## 7. Execution gate

For non-trivial work:
- approve slice plan first
- execute only approved slices
- capture extra ideas as backlog candidates

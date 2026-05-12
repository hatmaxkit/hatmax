# Slice Delivery

## Slice Workflow

1. Evaluate single-slice vs multi-slice execution.
2. Define behavior-first slice boundaries.
3. Implement and validate per slice.
4. Integrate incrementally while keeping repository healthy.

## Naming Convention

1. Use `Slice 1`, `Slice 2`, `Slice 3`, ... for slice identifiers.
2. Use `T1.1`, `T1.2`, `T2.1`, ... for task identifiers.
3. Do not use letter-based or Roman numeral variants for slices/tasks.

## PR Title

- Required: `feat(slice-<n>): <short slice name>`

## PR Body

```md
## Summary

- <functional increment 1>
- <functional increment 2>

## Changes

- <path/to/file>
  - <change detail>
- <path/to/other_file>
  - <change detail>

## Validation

- <command and result>
```

## Ordering Rules

1. `Summary`: primary behavior change first.
2. `Changes`: runtime/library code, tests, docs, ops artifacts.
3. `Validation`: targeted to broad.
4. Report validation outcomes as `pass`/`fail`.

## Traceability Contract

- Keep explicit correlation across:
  - `ops/plans/*-plan.md`
  - `ops/trackers/*-tracker.md`
  - branch name and PR title
  - commit list mapped to slice tasks

## Blocker Handling

1. Capture failing command and exact error.
2. Attempt least-invasive compliant alternative.
3. Report blocker and stop risky changes.

## Completion Checklist

1. Scope implemented.
2. Relevant checks run.
3. Tracker/plan status updated.
4. Final status reported with concrete outcomes.

## Post-Slice Tuning

1. After planned slices and automated checks pass, decide whether tuning is needed.
2. Tuning slices are recommended, not mandatory.
3. If needed, run one or more short tuning slices (UX polish, manual verification findings, integration adjustments).
4. Keep the same delivery discipline: one branch + one PR per tuning slice, base `dev`.
5. Record the decision (`tuning needed: yes/no`) and evidence in the tracker.

## Main Alignment

1. Slice PRs target `dev`, not `main`.
2. Never work on `main` directly.
3. Align `dev -> main` only through the release-alignment playbook.
4. The `main` update is a PR merge using rebase + fast-forward from `dev`.

## Mandatory Execution Rules

1. Execute one slice at a time.
2. Implement tasks in order.
3. Create one conventional commit per task.
4. Open one PR per slice.
5. Do not open PRs per task or per commit.
6. Complete PR review.
7. Merge the approved slice PR into `dev`.
8. Report implementation and validation status.
9. Start next slice only after previous slice is fully closed.

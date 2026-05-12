# Planned Delivery Workflow

Use this workflow for large features, architectural changes, or multi-slice workstreams.

1. Start from a long-form idea/ticket.
2. Create `ops/plans/*` and `ops/trackers/*` artifacts.
3. Define slices as behavior increments; define tasks inside each slice.
4. Implement tasks in order; create one Conventional Commit per task.
5. Open one PR per slice, targeting `dev`.
6. Complete review.
7. Report slice completion as: `Slice N of M completed`.
8. Start the next slice only after previous slice is merged.
9. After the final slice, publish a canonical end-to-end cycle report summarizing all slices.
10. Align `dev -> main` only through the release-alignment playbook when explicitly requested.

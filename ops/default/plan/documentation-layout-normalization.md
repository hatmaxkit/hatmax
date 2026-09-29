# Documentation Layout Normalization Delivery Plan

Status: Approved
Delivery set: documentation-layout-normalization
Slice strategy: behavior-first
Reason: the repository and generator must expose one documentation layout at
the same time. A single end-to-end slice keeps checked-in documentation,
validation, specifications, planning, rendering, conformance, and tests on the
same `README.md` contract without a transitional dual-layout state.
Ticket: `ops/default/ticket/ready/20260929091830-normalize-documentation-entrypoints-and-media.md`
Spec: `ops/default/spec/boxed-diataxis-documentation.md`
Tracker: `ops/default/tracker/documentation-layout-normalization.md`
Base branch: `dev`
Planning base: `73fcc8c076b30259b522da8fbb056bf6ae123da1`
Active slice: Slice 1
Execution gate: satisfied when this approved plan and tracker are committed to
`dev`

## Objective

Adopt forge-native `README.md` entrypoints throughout the Diataxis tree and
move documentation-owned images to `assets/img/docs/`. Apply the same layout
to the Hatmax generator so manually maintained and generated documentation
remain one canonical system.

The migration must preserve documentation meaning, link reachability,
generator ownership boundaries, deterministic plans, and conformance checks.
It does not rewrite User Guide content; the separate
`user-guide-technical-journey` delivery set follows this migration.

## Layout Contract

The top-level shape is:

```text
docs/
├── README.md
├── tutorials/
├── how-to/
├── reference/
└── explanation/

assets/
└── img/
    └── docs/
        ├── brand/
        └── gallery/
```

Use `README.md` for every directory entrypoint under `docs/`:

- documentation root;
- quadrant root;
- User Guide root;
- reference, how-to, explanation, and tutorial subject directories;
- generated Diataxis subject directories.

Keep descriptive filenames for ordinary chapters that do not own a directory.
Do not keep parallel `index.md` compatibility files. Relative links and
directory shorthand must resolve to the canonical `README.md` files.

Use `assets/img/docs/` for documentation-owned images. Use
`assets/img/docs/brand/` for the hero, logo, and social image, and
`assets/img/docs/gallery/` for gallery screenshots. Diataxis directories do
not own shared binary assets.

## Migration Boundaries

Update current contracts and behavior:

- checked-in documentation paths and links;
- root and package README links;
- `scripts/docs-check.sh` required entrypoints and top-level rule;
- accepted generator specifications that define the current layout;
- generator project inspection and Markdown-link resolution;
- deterministic documentation target and index planning;
- generated backlinks and index links;
- documentation rendering and conformance;
- CLI, interaction, planning, execution, and acceptance tests.

Do not rewrite delivered plans and reports merely to erase historical paths.
They remain records of the layout that existed when those delivery sets ran.
Update an operational artifact only when it is an active contract or required
by the current delivery.

Do not rewrite User Guide prose beyond link and path changes required by this
migration. Do not start the technical-journey content rewrite in this set.

## Ordered Slices

| Slice | Short Name | Branch | Pull Request Title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | canonical README layout | `feat/documentation-readme-layout` | `feat(slice-1): adopt forge-native documentation layout` | `ops/default/report/slices/documentation-layout-normalization/slice-1-canonical-readme-layout.md` |

## Slice 1: Canonical README Layout

### Purpose

Migrate the complete repository and generator documentation contract in one
reviewable change. A project inspected or mutated after this slice uses only
`README.md` directory entrypoints and documentation media under
`assets/img/docs/`.

### Tasks

- T1.1: Rename checked-in documentation entrypoints, move documentation image
  assets, update all live links, and update `scripts/docs-check.sh`.
- T1.2: Update accepted documentation and generator specifications plus Book
  rules to define the canonical `README.md` layout.
- T1.3: Update generator inspection, planning, rendering, conformance, CLI
  reporting, and all affected tests to use `README.md` paths.
- T1.4: Validate the integrated behavior, update the ticket and tracker, and
  publish the slice report.

### Expected Commits

| Task | Commit |
| --- | --- |
| T1.1 | `docs: adopt forge-native documentation entrypoints` |
| T1.2 | `docs(generator): adopt the README documentation contract` |
| T1.3 | `feat(generator): render README documentation entrypoints` |
| T1.4 | `docs(ops): report documentation layout normalization` |

### Validation

- `make docs-check`
- `go test ./generator/... ./internal/hatmaxcli/...`
- `make lint-strict`
- Verify the top level of `docs/` contains only `README.md` and the four
  Diataxis directories.
- Verify no live documentation, validation script, accepted specification,
  generator source, Book rule, or test expects a documentation `index.md`.
- `git diff --check`

## Delivery-Set Gate

After Slice 1 is merged into `dev`, run `make check` against the exact
integrated commit. Close the delivery set only when that gate passes and the
canonical Forgejo repository renders the directory README files as landing
pages.

## Prerequisites

- The ticket is ready with the `README.md` and `assets/img/docs/` decision.
- This plan and its tracker are approved and committed on `dev`.
- Slice 1 starts from the exact integrated planning commit on `dev`.
- The User Guide technical-journey delivery set does not start until this set
  is delivered and closed.

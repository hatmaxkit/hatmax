---
id: TKT-20260929091830
title: Normalize documentation entrypoints and image assets
status: solved
kind: task
severity: unclassified
priority: unclassified
scope: docs
tags: documentation-layout, forge-rendering, media-assets
source: chat
reported_at: 2026-09-29T09:18:30Z
ready_at: 2026-09-29T09:27:33Z
started_at: 2026-09-29T09:29:13Z
reviewed_at: 2026-09-29T09:32:53Z
closed_at: 2026-09-29T09:40:43Z
resolution: fixed
branch: feat/documentation-readme-layout
pr: 47
commits: a854c2750876, 92320c0fab6c, 6ba9c6be5185, b6225d7ef17f
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->


## Observed Behavior

Hatmax uses `index.md` for the documentation root, quadrant indexes, User Guide
entrypoint, and subject pages. This is structurally consistent, but Forgejo and
GitHub do not render those files automatically when a reader opens their
containing directory. The reader must select each index explicitly.

The top level of `docs/` also contains `img/` beside the documentation index
and the four Diataxis quadrants. The images include shared repository
presentation assets and gallery screenshots, so they do not have one natural
Diataxis reader intent.

The `index.md` convention is embedded beyond Markdown links. It appears in the
documentation checker, accepted generator specifications, generator planning,
rendering and conformance behavior, and their tests. Renaming only the current
documents would make generated documentation disagree with the repository
layout.

## Expected Outcome

Apply one repository-wide documentation layout:

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
```

Use `README.md` as every directory entrypoint under `docs/`, including the
documentation root, quadrant roots, the User Guide root, and subject
directories. Keep descriptive filenames for ordinary chapters that are not
directory entrypoints. This gives Forgejo and GitHub an automatically rendered
landing page at every navigable directory.

Move shared documentation and repository images from `docs/img/` to
`assets/img/docs/`. Reserve that subtree for documentation-owned images and
retain meaningful groups beneath it, such as `brand/` and `gallery/`. Do not
place shared images under a Diataxis quadrant: Diataxis classifies reader
needs, not media file types. A future documentation publisher must explicitly
include the repository-level documentation assets.

Treat the migration as a product contract change rather than a Markdown-only
rename. Update all affected documentation links, README files, validation
scripts, accepted specifications, generator path planning, rendering,
conformance, project inspection, and tests in the same coordinated delivery.
Do not mix this migration into the User Guide technical-journey delivery set.

## Validation

- Opening `docs/` and every documentation subject directory in Forgejo or
  GitHub automatically renders its `README.md` entrypoint.
- The top level of `docs/` contains only `README.md` and the four Diataxis
  quadrant directories.
- All shared documentation images live under `assets/img/docs/`, and every
  image reference resolves from the root README and documentation pages.
- No active documentation, tool, specification, generator contract, or test
  expects a documentation `index.md` path.
- Generated Diataxis documents use the same `README.md` convention as the
  repository documentation.
- `make docs-check`, the focused generator test suites, and
  `git diff --check` pass.

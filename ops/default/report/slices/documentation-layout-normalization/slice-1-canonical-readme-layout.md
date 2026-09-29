# Slice 1: Canonical README Layout

Status: reviewing
Delivery set: documentation-layout-normalization
Plan: [Documentation Layout Normalization Plan](../../../plan/documentation-layout-normalization.md)
Tracker: [Documentation Layout Normalization Tracker](../../../tracker/documentation-layout-normalization.md)
Branch: `feat/documentation-readme-layout`
PR: #47

## Purpose

Make every Hatmax documentation directory directly readable on Forgejo and
GitHub while preserving one canonical layout for maintained and generated
Diataxis documentation.

## Delivered Behavior

Every directory under `docs/` now uses `README.md` as its entrypoint. The
documentation root contains only that entrypoint and the four Diataxis
quadrants. Shared documentation images now live under
`assets/img/docs/brand/` and `assets/img/docs/gallery/`.

The documentation gate enforces the top-level shape, requires a `README.md`
in every documentation directory, checks ordinary Markdown links, and checks
HTML image sources. Root, package, example, guide, gallery, and quadrant links
use the new paths.

The generator now inspects, plans, renders, reports, resolves, and checks
`README.md` targets. Generated subject pages link back to quadrant README
entrypoints, and root and quadrant navigation effects use the same convention.

## Implementation Notes

The migration is intentionally atomic. It does not retain parallel `index.md`
files or a compatibility mode. Historical delivered plans and reports retain
their original paths, but current specifications, Book rules, runtime behavior,
and tests use only the new contract.

User Guide prose was not rewritten beyond the links required by this layout
change. The separate technical-journey delivery set remains responsible for
its editorial replacement.

## Contracts Added or Changed

- `README.md` is the canonical entrypoint for every documentation directory.
- `docs/` contains only its README and the four Diataxis quadrant directories.
- Documentation-owned images live under `assets/img/docs/`.
- Generated Diataxis targets and navigation use exact `README.md` paths.
- Directory-style generated links resolve to the directory's `README.md`.

## Files of Interest

- `docs/README.md`
- `assets/img/docs/`
- `scripts/docs-check.sh`
- `ops/default/spec/boxed-diataxis-documentation.md`
- `generator/project/documentation.go`
- `generator/plan/documentation.go`
- `generator/execute/render_documentation.go`
- `generator/execute/conformance_documentation.go`

## Validation

- `make docs-check` passed.
- `go test ./generator/... ./internal/hatmaxcli/...` passed.
- `make lint-strict` passed with zero issues.
- Documentation top-level, per-directory README, and active `index.md` path
  audits passed.
- `git diff --check` passed.

The full repository gate is deferred until the exact slice commit is merged
into `dev`, as required by the delivery-set plan.

## Risks and Follow-ups

A future documentation publisher must include repository-level assets from
`assets/img/docs/`. No publisher is configured by this slice.

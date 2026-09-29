# Hatmax Validation and Publication

Hatmax has no `staging` branch and no nightly workflow. These steps replace
the global nightly `main` integration for this repository.

## Repository authority and backup mirrors

1. `origin` on Forgejo is the only operational remote and the source of truth
   for branches, tags, pull requests, CI, and releases.
2. `mirrorcb` on Codeberg and `mirrorgh` on GitHub are download-only backup
   mirrors. Do not use them for integration, pull requests, CI, release
   identity, or repository-state decisions.
3. Fetch branches and tags only from Forgejo with `git fetch origin --prune
   --tags`. Inspect backup refs with `git ls-remote` when publication needs
   verification. Do not use `git fetch --all --tags` as a release operation.
4. Configure both backup remotes with `skipDefaultUpdate = true`, `tagOpt =
   --no-tags`, and a fetch refspec limited to `main`. This keeps routine
   `git fetch --all` operations authoritative and prevents backup-only tags or
   branches from entering the local namespace.
5. Publish only the verified `origin/main` ref and explicitly approved version
   tags to both backups. Never publish `dev`, feature branches, pull-request
   refs, or local-only tags.
6. A version tag is created and verified on Forgejo first, then the exact same
   tag ref is pushed outward to both backups.

## Focused development

1. Run only the smallest relevant check while implementing.
2. Use `make test` for behavior covered by the test suite.
3. Use `make vet` when a change needs `go vet`.
4. Use `make lint-strict` for formatting and lint. It checks `gofmt` on
   tracked Go files, then runs `golangci-lint` with `nlreturn`,
   `noinlineerr`, and `wsl_v5`. The lint cache is `.tmp/lint/gocache`.
5. Leave `make ci` for badge publication. That target rewrites
   `.badges/ci.json` and `.badges/coverage.json`.
6. `.githooks/pre-commit` formats staged Go files and lints their directories
   after `make install-hooks` points Git at `.githooks/`.

## Local aggregate gate

1. Run `make check` before a release-alignment pull request that changes
   runtime behavior.
2. `make check` runs `format`, `vet`, `test`, `test-coverage-check`, and
   `lint-strict`.
3. `test-coverage-check` fails when total coverage is below 80%.

## CI

1. `.github/workflows/ci.yml` runs on pushes to `main` and on pull requests
   targeting `main`.
2. The lint job runs `make lint-strict`.
3. The test job runs `make test-coverage-profile` against Postgres 16 and
   fails when coverage is below 80%.
4. CI does not rewrite `.badges/`.

## Badges

1. `README.md` publishes `.badges/ci.json` and `.badges/coverage.json`.
2. Refresh the coverage badge with `make update-badge`.
3. Refresh both badges with `make ci`. That target runs `format`, `vet`,
   `test`, `test-coverage-100`, and `lint-strict`, writes the CI badge, and
   then runs `update-badge`.
4. `make test-coverage-100` fails when coverage is below 70%.
5. Commit refreshed badge files when a release-alignment candidate has made
   them stale.

## Release alignment

1. Open a pull request for every `dev` to `main` alignment. The pull request
   is required for docs-only changes as well.
2. Run the local aggregate gate unless the docs-only exception below applies.
3. Use the title `chore(release): align dev with main`.
4. Use this pull-request body:

```md
## Summary

- align `main` with the latest merged state from `dev`
- include all previously reviewed and merged work

## Validation

- make check (pass)
```

5. Merge `dev` into `main` only through that pull request, using rebase and
   fast-forward.
6. Completing work on `dev` does not authorize the alignment. The maintainer
   controls the merge.
7. After the merge is verified on Forgejo, publish the exact `origin/main` ref
   to `mirrorcb/main` and `mirrorgh/main`. Publish a version tag only after the
   separate release gate authorizes it.

## Docs-only alignment

When the `dev` to `main` diff contains only Markdown repository documentation:

1. Skip runtime tests.
2. Run `make docs-check` to validate the Diataxis structure, local links,
   example compilation, stale import paths, and whitespace.
3. Use this validation section instead:

```md
## Validation

- make docs-check (pass)
- local aggregate gate skipped by the docs-only exception
```

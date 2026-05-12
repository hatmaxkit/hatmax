# Dev-to-Main Release Alignment

Canonical release-alignment PR from `dev` to `main`.

1. Open a PR for every `dev -> main` alignment. The PR is mandatory, including docs-only changes.
2. Run the configured release-alignment gate.
3. Docs-only gate exception: if the diff contains only Markdown repository documentation files, skip runtime tests and validate the docs-only diff plus whitespace.
4. Open PR with title: `chore(release): align dev with main`.
5. Use this PR body:

```md
## Summary

- align `main` with the latest merged state from `dev`
- include all previously reviewed and merged slice/ticket work

## Validation

- <release-alignment gate command> (pass)
```

For docs-only alignment PRs, use this validation section instead:

```md
## Validation

- docs-only diff verified
- whitespace/diff validation (pass)
- release-alignment gate skipped by docs-only exception
```

6. Merge `dev -> main` only through PR, using rebase + fast-forward.

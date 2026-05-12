# Linting and Quality Gates

This repository uses the lightest quality gate that still protects changed behavior.

## Baseline checks

- Syntax, type, compile, or static checks for changed files when the stack provides them.
- Targeted tests for changed behavior.
- Broader test or integration commands before merge when risk or shared behavior justifies them.

## Optional stricter checks

If enabled by project setup:
- formatter checks
- lint/static analysis checks
- type checks
- coverage checks

## Hook policy

If git hooks are configured, pre-commit should run fast checks only:
- changed-file syntax checks
- targeted tests where possible

Full suite should run before merge.

## Rules

- Do not skip tests for behavior changes.
- Fix failing checks before declaring work complete.
- Report validation commands and outcomes explicitly.

## Project-specific commands

- Keep canonical validation commands in project documentation or tooling.
- Prefer commands that can run from the repository root without hidden local setup.

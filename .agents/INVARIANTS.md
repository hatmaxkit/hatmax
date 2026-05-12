# Collaboration Invariants

This document defines non-negotiable execution invariants.

## 1. Scope Integrity

- Execute only approved scope.
- Out-of-scope ideas must be recorded, not implemented.

## 2. Explicit Approval Gates

- High-impact changes require explicit approval before implementation.
- Plan execution starts only after plan approval.

## 3. Small Safe Steps

- Prefer incremental, reversible changes.
- Keep repository in a working state between steps.

## 4. Quality Gate Before Closure

- Do not mark work complete with failing checks.
- Run relevant tests and report outcomes.

## 5. Git Safety

- No destructive Git operations without explicit approval.
- Keep history readable and scoped.

## 6. Documentation Separation

- Product specifications belong in `docs/`.
- Working rules belong in `AGENTS.md` and `.agents/*`.

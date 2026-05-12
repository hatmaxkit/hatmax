# Engineering Conventions

This document defines repository-wide engineering rules.

## Dependencies

- Prefer the platform, standard library, framework, and tooling already in use.
- Ask before introducing new gems, packages, services, or build tools.
- Do not add dependencies to avoid small, local design decisions.

## Code style

- Keep methods short and explicit.
- Use comments only for non-obvious logic.
- Prefer deterministic behavior in core flows.
- Name objects after the responsibility they own, not the implementation trick they use.

## Boundaries

- Put behavior in the established app, service, model, library, adapter, or UI boundary for that subsystem.
- Keep entrypoints, controllers, command handlers, and scripts thin.
- Separate IO/adapters from decision logic.
- Keep persistence, rendering, orchestration, and external integrations in distinct units.

## Testing

- Behavior changes must include appropriate validation.
- Add or update tests when behavior, contracts, persistence, or workflow boundaries change.
- Favor table-driven style where inputs/outputs are explicit.
- Avoid fragile tests coupled to incidental formatting.
- Match test scope to risk: unit tests for local rules, integration tests for workflow contracts.

## Anti-patterns

Avoid:
- hidden global mutable state
- mixing unrelated concerns in one file or commit
- speculative abstractions without active use
- broad rewrites that do not reduce current complexity

Prefer:
- explicit contracts
- small cohesive units
- incremental safe changes
- local consistency over generic architecture

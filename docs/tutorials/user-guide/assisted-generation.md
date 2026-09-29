# Assisted Generation

Hatmax includes an experimental AI-interpreted generator for changing an
existing compatible Hatmax application. It does not create the application,
replace the architecture described by this guide, or act as a general coding
harness.

The current product surface is a CLI:

```sh
hatmax generate "Create an invoice feature with a required number."
```

A conversational TUI is planned but not yet delivered. The generator may
answer limited interaction needs around its workflow, but implementation
requests remain constrained to Hatmax features and primitives.

## Understand the Control Boundary

Codex interprets natural language into a bounded typed intent. Hatmax then
selects the canonical archetype, expands all required surfaces, validates the
intent, and presents a sealed plan. The model does not choose arbitrary
libraries, architecture, files, or commands.

No project mutation occurs until the user explicitly approves that plan. If
the project changes after planning, Hatmax rejects the stale approval. After
approval, Hatmax-owned renderers apply the edits, check structural conformance,
and run applicable commands discovered from the project.

The generator currently supports creating a canonical server-rendered CRUD
feature, adding a field, adding a Hatmax validation rule, and explicitly
requested boxed Diataxis documentation. Documentation is never added merely
because an implementation changed.

## Inspect Before and After

Before approval, confirm the intent, capabilities, affected surfaces,
documentation mode, allowed effects, and validation commands. A required
product decision should appear as a clarification, not as an implementation
guess.

After completion, review the Git diff with the manual application model from
this guide:

```text
model -> store -> service -> handler -> templates -> wiring -> tests
```

Verify project-specific behavior even when the generated conformance and
command gates pass. Assisted generation reduces mechanical assembly; it does
not own product decisions, deployment acceptance, or code review.

Requests for alternate databases, ORMs, routers, validation frameworks, or a
client-side application state model are outside the current Hatmax Book. Use a
general coding tool outside Hatmax when the desired result is not a Hatmax
application change.

For installation requirements, supported operations, runtime behavior,
approval rules, diagnostics, and exit statuses, use the
[Generator Reference](../../reference/generator/README.md).

---

[Previous: Testing and Evolution](testing-and-evolution.md) ·
[User Guide](README.md)

# Workspace Architecture

This document describes the workspace structure and ownership boundaries.

## Baseline Structure

```text
AGENTS.md      # top-level agent operating contract
.agents/       # collaboration and execution rules
docs/          # durable user-facing or technical documentation, when present
ops/           # specs, plans, tickets, quiz notes, dumps, and delivery artifacts, when present
```

## Boundary Rules

- Keep working rules in `AGENTS.md` and `.agents/*`.
- Keep durable product or technical documentation in `docs/`.
- Keep delivery records, planning artifacts, quiz notes, dumps, and other conversation captures in `ops/` when the workspace uses that structure.
- Put implementation behavior in the existing application, package, library, service, or script boundaries already present in the workspace.
- Do not introduce top-level directories or framework layers without clear need.

## Change Discipline

- Structural changes must be explicit in the requested scope.
- Prefer extending established local structure before creating new architectural categories.
- Record project-specific exceptions in this file instead of changing the generic protocol rules.

# Generator

The `hatmax` command provides an interactive, AI-interpreted generator for
existing Hatmax applications. Hatmax owns the plan, renderers, mutations,
conformance checks, and validation. Codex is restricted to mapping natural
language into the bounded intent schema.

## Command

Run the command from the application root:

```text
hatmax generate "<request>"
```

The command accepts exactly one non-empty request argument. Any other command
shape prints the usage line and exits with status `2`.

## Requirements

- The target is a Go module with a supported Hatmax dependency. Book version
  `1` supports Hatmax versions from `v0.4.0` up to, but not including,
  `v0.6.0`.
- A compatible `codex` executable is available on `PATH` and authenticated.
  Installation and authentication are covered by the
  [official Codex CLI documentation](https://learn.chatgpt.com/docs/codex/cli).
- Project-owned generation, formatting, and validation tools required by the
  discovered commands are installed. A Postgres feature normally requires
  `sqlc`; the project's lint command may require `golangci-lint`.

## Supported Operations

| Operation | Effect |
| --- | --- |
| `create_feature` | Creates a canonical server-rendered CRUD feature. |
| `add_field` | Adds one field across the existing feature's required surfaces. |
| `add_validation` | Adds a Hatmax-owned validation rule and its required coverage. |
| `document_feature` | Documents inspected existing behavior without implementation changes. |

Implementation requests use the server-rendered CRUD archetype and Book-owned
capabilities for Postgres persistence, Hatmax validation, and HTMX forms.
Requests for alternate databases, ORMs, routers, validation frameworks, or
client-side application state are outside this Book.

Documentation changes require explicit documentation intent. A request only
for documentation may affect only documentation. A combined request may
document the planned implementation. Generated pages use the selected
Diataxis quadrant, remain reachable from the documentation indexes, and
preserve text outside Hatmax-managed sections.

## Interaction Contract

One run performs these steps:

1. Inspect the module, repository state, layout, instructions, documentation,
   and recognized project commands.
2. Ask Codex for a schema-bounded interpretation. Project paths and file
   contents are not included in that request.
3. Collect any required clarifications, repeat interpretation with those
   explicit answers, and validate the typed intent.
4. Expand the intent through the Hatmax Book into a sealed, digest-bound plan.
5. Print the plan and require `y` or `yes` approval.
6. Reject approval if the project fingerprint has changed.
7. Render and apply only the effects authorized by the plan.
8. Check conformance and run the applicable project commands.
9. Print the execution report.

A negative answer or end of input cancels the plan. Approval is never inferred
from the original request.

## Codex Runtime

Hatmax locates `codex` on `PATH`, checks the CLI and managed App Server
versions, and reuses a compatible resident daemon. It starts the daemon only
when none is available and never stops or restarts it. Each command opens a
short-lived proxy and reuses the isolated thread assigned to that project;
closing the command does not stop the resident daemon.

## Exit Statuses

| Status | Meaning |
| ---: | --- |
| `0` | Generation completed. |
| `1` | Setup or unclassified interaction failure. |
| `2` | Invalid command usage. |
| `3` | Clarification or approval was cancelled. |
| `4` | Required product decisions remain unresolved. |
| `5` | The request is outside the Hatmax Book. |
| `6` | The typed intent failed deterministic validation. |
| `7` | Project drift invalidated the approved plan. |
| `8` | Rendering, mutation, conformance, or project validation failed. |

For a guided run, see
[Generate a Feature](../../tutorials/user-guide/generation.md).

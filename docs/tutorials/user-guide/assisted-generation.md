# Assisted Generation

This chapter uses `hm` to create a Hatmax application and then continue its
development through the same conversation. The outcome is a compiling
application whose generated structure follows the application model developed
throughout this guide.

Hatmax is conversational, but it is not a general coding harness. You can ask
ordinary questions or make informal remarks; they do not become project
changes. A mutation must resolve to a typed operation admitted by the Hatmax
Book, produce a visible plan, and receive explicit approval.

## Prepare the Builder

Install `hm` and authenticate the Codex CLI as described in the
[official Codex CLI documentation](https://learn.chatgpt.com/docs/codex/cli):

```sh
go install hatmax.adrianpk.com/cmd/hm@latest
```

The Codex CLI and its resident App Server must have compatible versions.
Hatmax reuses that resident process and never stops or restarts it.

Choose the parent directory in which the application should be created:

```sh
cd ~/Projects
hm
```

Running from the parent is intentional. Hatmax creates a normalized child
directory after it knows the application identity. Running `hm` from an
existing compatible Hatmax project instead opens that project's conversation.

## Create an Application

Enter a complete request in the composer and press `Enter`:

```text
Create a Hatmax application named Ledger with module path example.com/alex/ledger and an invoice feature with a required number string field.
```

A shorter request is also valid. Hatmax asks focused questions for required
identity it cannot derive; it does not require a description, niche, or
initial feature. PostgreSQL is part of the canonical Hatmax application and
does not need to be requested.

Inspect the displayed plan before approving it. An application plan includes
the scaffold as its first unit and each requested initial feature as a
dependent unit. It also lists exact effects, conformance rules, and validation
commands. No target file has changed yet.

Press `Ctrl+A` to approve the displayed digest. `Esc` cancels the proposal
without changing the target.

After approval, Hatmax creates `~/Projects/ledger`, validates the canonical
scaffold, compiles it, and runs applicable project checks. The conversation is
then rebound from the parent scope to the new project. This prevents the
pre-project backend thread from becoming hidden authority inside the created
application.

## Read the Result

A successful result reports `Completed`. If compilation and conformance pass
but an external test prerequisite is unavailable, the result is `Completed;
validation incomplete` and identifies the blocked command. A real generated
test failure is not downgraded to incomplete validation.

Review the generated application using the same composition model used for
manual development:

```text
main -> internal/application -> infrastructure -> features -> handlers -> templates
```

The composition root remains explicit, `main.go` contains only `main`, and
features retain the canonical vertical slice:

```text
model -> store -> service -> handler -> templates -> wiring -> tests
```

Generated files are ordinary Go, SQL, templates, and assets. The application
does not require the generator at runtime.

## Continue the Conversation

From the created project, run `hm` again:

```sh
cd ~/Projects/ledger
hm
```

The active project conversation resumes. You can now request another admitted
change, for example:

```text
Add an optional notes text field to invoice.
```

Hatmax reinspects the project before planning. Previous dialogue can preserve
product context, but source inspection remains authoritative. If the project
changes after a plan is displayed, approval returns `Plan stale`; send a
revised request or let Hatmax create a fresh plan.

Use these controls during the session:

- `Enter` sends a request and `Ctrl+J` inserts a newline;
- `Ctrl+A` approves the current plan;
- `Esc` cancels work or the current proposal;
- `Ctrl+N` starts a new conversation for the same scope;
- `Ctrl+H` expands key help;
- `Ctrl+C` exits safely.

## Select a Retained Conversation

Conversation state is stored outside the repository. List retained
conversations for the current project, select one, and reopen the TUI:

```sh
hm conversation list
hm conversation resume <conversation-id>
hm
```

To deliberately start over without changing source:

```sh
hm conversation new
hm
```

Resetting or losing local conversation state does not affect the project.
Hatmax reconstructs project facts from source and the compatible Book.

## Use the Headless Surface

For one focused request, use the same kernel without the TUI:

```sh
hm generate "Add a required issued-at timestamp to invoice."
```

The command prints the plan and accepts only `y` or `yes` as approval. It is
useful for automation and acceptance checks, while `hm` remains the primary
interactive surface.

The previous `hatmax` command is a temporary compatibility alias. New scripts
and documentation should use `hm`.

The [Generator Reference](../../reference/generator/README.md) defines the
complete command surface, supported operations, state locations, runtime
contract, exit statuses, and migration policy.

---

[Previous: Testing and Evolution](testing-and-evolution.md) ·
[User Guide](README.md)

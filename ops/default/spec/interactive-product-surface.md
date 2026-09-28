# Interactive Generator Product Surface

Status: Approved
Kind: Subordinate product-surface specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Intent: `ops/default/spec/intent-and-planning.md`
Execution: `ops/default/spec/execution-and-conformance.md`

## Purpose

This specification defines the first usable interactive surface for the
Hatmax generator and the boundary between that surface and an interpreter
backend. The first backend adapter uses a resident Codex App Server
authenticated through the user's ChatGPT subscription. Pi and direct API
adapters can implement the same boundary later, but they are not part of the
first product surface.

The model interprets user intent. Hatmax owns project inspection, context
selection, intent validation, planning, approval, execution, conformance, and
reporting. The interpreter backend never becomes an alternate implementation
engine.

## Why Hatmax Wraps Codex

Using Codex directly would expose a general coding agent whose valid solution
space includes many non-Hatmax architectures. A prompt can encourage Hatmax
style, but it cannot make canonical construction, complete obligation
expansion, or dependency policy deterministic.

Hatmax therefore wraps a headless interpreter backend with its own terminal UI
and control protocol. The initial Codex backend resolves natural-language
intent inside a bounded schema. Hatmax independently owns the allowed
vocabulary, Book context, deterministic plan, approval gate, renderers,
project mutation, conformance, and validation.

The wrapper is the product boundary. It prevents conversational flexibility
from becoming architectural flexibility. A user cannot escape the Hatmax Book
through phrasing, and Codex cannot replace a Hatmax primitive, omit an
obligation, or edit an undeclared surface.

## Product Boundary

The first product is a local terminal application backed by one resident Codex
App Server. A user starts one interaction for an existing Hatmax project with
a natural-language request:

```text
hatmax generate "Add an invoice feature"
```

The terminal UI connects to the resident server, reuses the isolated Codex
thread for the current project, and normally requires one model turn to
interpret the operation. It can ask focused clarification questions, print
the complete Hatmax plan, require explicit approval, apply the plan, and print
the execution report.

Starting a new command must not start a new Codex process when a compatible
managed App Server is already available. Process startup, authentication,
protocol initialization, and conversational context are reused across Hatmax
commands. Model inference still occurs for each new interpretation.

The initial surface supports the operations already admitted by the Book:

- `create_feature`;
- `add_field`;
- `add_validation`.

Documentation requests remain explicit. Until the boxed documentation
capability is delivered, the interactive surface must reject documentation
intent as unsupported instead of ignoring it or generating prose outside the
canonical Diataxis workflow.

## Runtime Topology

The initial runtime topology is:

```text
Hatmax TUI
    -> Codex App Server proxy
        -> managed Codex App Server daemon
            -> isolated thread for one Hatmax project
                -> one turn per request or clarification answer
```

Hatmax uses the Codex-managed local daemon and control socket when the
installed Codex version provides them. The TUI starts the daemon only when it
is absent. It must not restart, stop, or replace an already running compatible
daemon. An incompatible running daemon produces a precise version error and
requires explicit user action.

The TUI connects through the local App Server proxy and speaks the JSON-RPC
protocol. It does not expose a TCP or WebSocket listener. The short-lived proxy
belongs to one Hatmax client connection; the App Server process and project
threads survive that client process.

The daemon is shared by the logged-in user, but model context is not shared
between projects. Hatmax starts or resumes one Codex thread for each canonical
project identity and compatible Book version. The thread uses an isolated
Hatmax-owned context directory with no target-project files. Project identity
is represented by an opaque local identifier rather than repository contents
or credentials.

The project thread is an interpretation cache, not a source of truth. Hatmax
starts a new thread when the Book contract changes incompatibly, the adapter
schema changes, the authenticated Codex identity changes, the stored thread is
invalid, or the user explicitly resets the interaction. Losing a thread never
prevents project reconstruction from current source and Book state.

## Interaction Lifecycle

One interaction follows this state sequence:

1. Ensure that a compatible Codex App Server is available and connect through
   its local proxy.
2. Inspect the project and select the compatible Hatmax Book.
3. Start or resume the isolated Codex thread for the project and Book version.
4. Compile the user request and bounded context for the interpreter.
5. Start one schema-constrained turn and receive one typed intent,
   clarification result, or unsupported result.
6. Ask the user for each material clarification and continue the same thread.
7. Validate the intent and deterministically expand it into a sealed plan.
8. Print the plan before any project edit.
9. Require explicit approval or terminate without changes.
10. Reinspect the project and reject stale plans before execution.
11. Dispatch the approved operation to the canonical Hatmax renderer.
12. Apply the execution manifest, run conformance, and run sealed repository
    commands.
13. Print the final execution report or one precise failure result.

The lifecycle is explicit and inspectable. The command must not skip plan
presentation because the request appears simple or low risk.

A normal supported operation uses one Codex turn. Additional turns are
admitted only to answer a necessary clarification or recover once from a
rejected structured result. Hatmax never invokes the model once per file,
affected surface, validation step, or repair attempt.

## Conversation State

Hatmax owns the authoritative interaction state: current project inventory,
selected Book, request, clarification questions and answers, normalized
intent, sealed plan, and execution evidence.

Codex retains conversational context in the isolated project thread across
Hatmax commands. Hatmax still supplies current bounded project and Book
context with every turn. Prior model observations cannot override live
inspection, intent validation, deterministic planning, or conformance.

Codex stores and resumes its own thread history. Hatmax does not store
conversation, intent, or plan artifacts inside the generated project. The
thread is a discardable operational cache; current project source and Book
state remain authoritative. A future execution-log feature may persist
reports without changing that ownership rule.

## Interpreter Boundary

The existing `eval.Interpreter` interface remains the backend-neutral
boundary. The interpreter receives only an `eval.Request` extended with the
bounded clarification conversation required for the current interaction.

The request contains:

- the current user goal and explicit clarification answers;
- project fingerprint and compatible Hatmax version;
- existing canonical feature names;
- the selected Book version;
- supported archetypes, operations, capabilities, prerequisites, and
  unsupported variants.

The request does not contain arbitrary repository files, repository paths,
commands, credentials, unrelated agent sessions, or the complete Hatmax Book.

The only admitted interpreter results are:

- one typed Hatmax intent;
- focused clarification questions;
- structured unsupported diagnostics.

The interpreter cannot return a plan, edit, command, dependency, file path, or
approval decision. Hatmax validates every result independently.

## Codex App Server Adapter

The first interpreter backend adapter connects to the managed Codex App Server
and uses the saved Codex authentication established by the user through
ChatGPT. It does not read, copy, print, or persist authentication material.

The adapter must:

- verify compatibility with the installed Codex CLI and App Server protocol;
- start the managed daemon only when no compatible daemon is available;
- connect through the local control socket and App Server proxy;
- initialize one JSON-RPC connection for each Hatmax client;
- start or resume one isolated thread for the project and compatible Book;
- issue one `turn/start` request for each interpretation attempt;
- constrain every turn with the versioned Hatmax interpretation output schema;
- use an isolated Hatmax-owned context directory as the thread working
  directory and withhold the target-project path and contents;
- select the read-only sandbox, disable every tool and external context source
  exposed by the compatible protocol, and grant no dynamic tools;
- deny target-project writes, shell commands, network access, MCP servers, and
  permission escalation;
- reject App Server requests for commands, file access, tools, or permissions;
- interrupt the active turn on cancellation or timeout;
- bound retained events and error payloads;
- return stable diagnostics without exposing credentials, model reasoning, or
  unrelated App Server events.

The isolated context directory contains only the bounded interpreter request
and generated schema material. The initial adapter does not grant Codex access
to the target project and does not ask Codex to inspect or implement the
requested change. Hatmax performs inspection and execution; Codex performs
natural-language interpretation only.

The adapter treats any tool activity as a protocol violation and interrupts
the turn. This boundary prevents Hatmax from disclosing or authorizing access
to the target project. A shared App Server still runs as the logged-in user and
is not claimed as an operating-system confidentiality boundary for unrelated
host files.

The adapter records non-secret provenance needed to interpret an evaluation:

- adapter and protocol contract versions;
- Codex CLI and App Server versions;
- whether the daemon and project thread were reused;
- requested model when explicitly configured;
- whether the backend default model was used;
- turn result and bounded timing classification.

Hatmax does not select a different backend or fall back to an API key when
Codex is unavailable. Missing executable, missing authentication,
incompatible protocol, daemon startup failure, thread failure, timeout,
cancellation, invalid schema output, and failed turn are distinct failures.

## Structured Output Contract

The interpretation schema is versioned with the Hatmax generator. It is
derived from the Go interpretation and intent contracts and rejects unknown
fields. Exactly one result variant must be present.

Schema validation at the Codex boundary is necessary but not sufficient. The
existing intent schema and semantic validation run again after parsing. A
schema-valid result cannot add Book-owned architecture or bypass project
compatibility checks.

Malformed, incomplete, contradictory, or oversized output produces no plan
and no project changes.

## Plan Presentation and Approval

The command prints the canonical YAML projection of the sealed plan. The
projection includes the requested operation, archetype, feature, capabilities,
affected surfaces, documentation intent, selected rule IDs, and plan digest.

The first product requires an explicit affirmative answer for every plan. An
empty answer, negative answer, end of input, or non-interactive input without
an approval channel cancels the interaction without project changes.

Approval applies only to the displayed plan digest and source project
fingerprint. Any relevant project drift invalidates approval and requires a
new interpretation and plan.

The initial command has no automatic approval flag. Automation and risk-based
approval policies require a later specification revision.

## Execution Coordination

A backend-neutral coordinator owns the complete operation after plan
approval. It selects the renderer from the admitted operation, prepares the
execution manifest, stages all declared mutations, commits them atomically,
reinspects the project, evaluates conformance, and runs only the repository
commands sealed in the manifest.

The coordinator does not ask the model to repair source. A renderer,
conformance, or validation failure terminates the interaction with exact
diagnostics. Failures before mutation leave the project unchanged. Atomic
commit failures use the existing rollback behavior. A failure after a
successful source commit reports every retained change and validation result;
it does not conceal the failed state or create a Git commit.

The command never commits, branches, pushes, opens a pull request, applies a
migration to a live database, deploys, or publishes. Those operations require
separate user authorization outside the generator.

## User-Facing Results

The command returns one of these observable outcomes:

- `clarification_required`, with one or more focused questions;
- `unsupported`, with stable Book or capability diagnostics;
- `cancelled`, with no project changes;
- `plan_stale`, with classified project drift;
- `execution_failed`, with mutation, conformance, or command evidence;
- `completed`, with the final execution report.

Human-readable output is the first interface. Stable internal result types
must not depend on terminal wording, color, or interactive formatting.

The completed result reports:

- admitted intent and plan digest;
- changed and unchanged surfaces;
- Book and Hatmax versions;
- interpreter-backend provenance without authentication data;
- conformance diagnostics;
- exact repository commands and exit results;
- warnings and unresolved external validation.

## Failure and Security Boundaries

The surface must fail without project changes when:

- project inspection or Book selection fails;
- the Codex App Server is unavailable, incompatible, or not authenticated;
- the isolated project thread cannot be started or resumed;
- interpretation times out or is cancelled;
- Codex returns invalid or unsupported structured output;
- material ambiguity remains;
- intent validation or plan expansion fails;
- the user rejects the plan;
- the project drifts before execution.

The adapter must not expose repository contents beyond the bounded
`eval.Request`. It must not include environment variables, Git credentials,
Codex authentication files, agent configuration, hidden reasoning, or raw
event streams in reports. It must not honor App Server requests to inspect the
target repository, execute commands, invoke tools, access the network, or
elevate permissions.

## Interpreter Backend Extensibility

Codex is the first backend adapter, not a special authority. Backend-specific
process and connection management, authentication detection, thread handling,
schema transport, and error mapping stay inside the adapter. Conversation,
planning, approval, execution, and reporting remain backend-neutral.

A later Pi adapter must satisfy the same `eval.Interpreter` and provenance
contracts. It cannot change intent semantics, plan structure, approval policy,
or execution behavior.

## Acceptance Criteria

- A user can start a supported Hatmax operation with one natural-language
  command.
- Hatmax reuses a compatible resident Codex App Server across commands instead
  of starting one Codex process for every request.
- Each project and compatible Book version uses an isolated Codex thread.
- A normal supported operation requires one schema-constrained Codex turn,
  regardless of the number of affected project surfaces.
- Codex uses the user's existing ChatGPT-managed CLI authentication without an
  application API key.
- Codex receives only bounded project and Book context; Hatmax withholds the
  target-project path, grants no tools, and cannot edit the project through the
  adapter.
- Every Codex result conforms to the versioned interpretation schema and is
  independently validated by Hatmax.
- Clarification answers continue the bounded project thread while Hatmax
  retains authoritative interaction state.
- Every admitted request produces a visible sealed plan before editing.
- Execution requires explicit approval of the displayed plan digest.
- Rejection, ambiguity, invalid model output, and relevant drift produce no
  project changes.
- The coordinator supports `create_feature`, `add_field`, and
  `add_validation` through the existing canonical renderers.
- Successful execution ends with Hatmax conformance and exact repository
  validation evidence.
- Backend or validation failures are distinct, bounded, and do not expose
  authentication data or hidden model reasoning.
- The interactive command performs no Git, publication, deployment, or live
  database operation.
- A future Pi adapter can implement the same backend-neutral boundary without
  changing the interaction lifecycle.

## Initial Configuration Decisions

- The initial command uses the backend default model and records both
  `backend_default` and the effective model reported by App Server.
- Each interpretation turn has a two-minute default timeout. User cancellation
  interrupts the active turn immediately.
- A structural application failure uses the existing atomic rollback. A
  conformance or repository-validation failure after a successful source
  commit retains the bounded changes for inspection and reports every change
  and failed check. Hatmax does not create a Git commit.

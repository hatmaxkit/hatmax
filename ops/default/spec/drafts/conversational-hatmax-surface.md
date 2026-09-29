# Conversational Hatmax Product Surface

Status: Draft
Kind: Subordinate product-surface specification
Umbrella: `ops/default/spec/interactive-hatmax-generator.md`
Current surface: `ops/default/spec/interactive-product-surface.md`
Intent: `ops/default/spec/intent-and-planning.md`
Execution: `ops/default/spec/execution-and-conformance.md`
Scaffold: `ops/default/spec/drafts/canonical-application-scaffold.md`
State model: `ops/default/spec/drafts/conversational-hatmax-surface-model.md`
Discovery: `ops/default/quiz/00001-hatmax-conversational-application-generation/artifact.md`

## Purpose

This specification defines Hatmax as a conversational terminal product for
creating and evolving Hatmax applications. It expands the delivered
single-request CLI into a persistent TUI without turning Hatmax into a general
coding harness.

The product is called **Hatmax**. Its user-facing description is a
conversational Hatmax application builder. `Generator engine` remains an
internal component name rather than the identity of the complete product.

Until this draft is accepted and delivered, the approved
`interactive-product-surface.md` contract remains authoritative.

## Product Boundary

Conversation is open; project mutation is closed.

The user may ask ordinary questions, make informal remarks, or discuss ideas
that do not produce code. Hatmax may respond conversationally. A conversation
turn acquires no mutation authority merely because it mentions software.

Only a request admitted by the Hatmax Book can enter the mutation lifecycle.
Hatmax must reject attempts to generate a non-Hatmax application, substitute a
Hatmax primitive, introduce an unapproved architecture, or use the TUI as an
unrestricted coding agent.

The product can eventually admit Hatmax-native maintenance capabilities beyond
initial generation, such as tests or fixtures. Each capability requires typed
intent, Book obligations, planning, conformance, and validation. Examples do
not implicitly expand the capability set.

## Command Surface

The canonical executable is `hm`.

```text
hm
```

Running `hm` opens the conversational TUI in the current directory. The same
executable owns headless subcommands for automation and tests. The existing
`hatmax` executable remains a temporary compatibility alias; it does not become
a second product with different authority.

Command availability and the compatibility lifetime must be verified before
implementation.

## Session Modes

Hatmax recognizes session context before the first mutation proposal:

- a parent directory can host a new application request;
- a compatible Hatmax project enters application-evolution mode;
- an incompatible project permits conversation but rejects project mutation
  with a precise diagnostic.

The user does not manually select a general chat mode or generator mode. Hatmax
classifies each conversational turn and makes any transition into project work
visible.

## Interaction States

The TUI exposes these semantic states independent of screen layout:

```text
conversation
  -> candidate_change
  -> clarification_required
  -> plan_ready
  -> awaiting_approval
  -> executing
  -> completed | validation_incomplete | failed | cancelled | stale
```

`conversation` accepts informal dialogue without project effects.
`candidate_change` tells the user that Hatmax recognized possible Hatmax work.
No edit occurs in either state.

Clarification, planning, approval, and execution reuse the deterministic
contracts already owned by Hatmax. TUI wording, animation, or layout never
becomes an alternate state authority.

## Adaptive Discovery

Hatmax accepts short prompts and detailed product descriptions. It collects
relevant facts from the conversation and asks one focused question at a time
only while required information is missing.

The model may select natural wording and identify candidate missing decisions.
The typed intent schema and semantic validator decide whether the request is
plannable. The model cannot declare its own output sufficient.

Once the required data exists, Hatmax stops asking optional enrichment
questions and presents the plan. The user can continue the conversation to
revise or enrich that plan before approval.

## Conversation and Mutation Separation

The interpreter result vocabulary adds a bounded conversational response that
cannot contain executable authority. The admitted result classes become:

- conversational response;
- candidate Hatmax intent;
- focused clarification;
- structured unsupported diagnostic.

Conversational text cannot specify paths, commands, dependencies, approvals,
or edits. If prior discussion contains product decisions relevant to a later
mutation, Hatmax surfaces those decisions in the typed intent and visible plan.
The user approves the explicit plan, not hidden conversational implications.

The Codex backend remains tool-less and cannot inspect or edit the target
project. Hatmax owns inventory, Book context, planning, execution, and
conformance.

## Resident Backend

The TUI connects to the compatible managed Codex App Server already used by
the delivered CLI. It reuses the resident daemon and isolated Hatmax thread
instead of launching a new Codex process for every message.

The daemon may be shared by the logged-in user, but Hatmax isolates context by
project or pre-project session identity and compatible Book contract. Missing
or incompatible backend state produces explicit diagnostics; Hatmax does not
silently switch providers or API credentials.

## Plan Presentation and Approval

Every mutation has a plan and explicit approval. The plan is available for
inspection before any edit.

Presentation is provisionally proportional to scope:

- a routine single operation uses a compact inline projection of intent,
  affected surfaces, and validation;
- application bootstrap or another composite request uses an expanded plan
  grouped by operation unit.

This difference changes presentation, not plan completeness or authority. The
user can revise, add, or remove units conversationally before approval. A
composite initial plan receives one approval and does not interrupt execution
with repeated prompts between units.

Adaptive plan presentation is a provisional usability decision. The first TUI
prototype must test whether it remains clear without adding ceremony.

## Persistent Project Conversation

Reopening `hm` in the same project resumes its prior local conversation by
default. Hatmax stores that context outside the project and never requires it
to build or run the application.

On resume, Hatmax:

1. resolves the local project or pre-project session identity;
2. reconnects to or replaces the compatible isolated interpreter thread;
3. reinspects the current source or target state;
4. invalidates stale pending intent and plans;
5. presents enough recent state for the user to continue.

Conversation improves continuity but never replaces source inspection. No
stored approval survives process exit, cancellation, drift, backend identity
change, or plan invalidation.

Hatmax must provide a deliberate way to start a fresh conversation without
deleting or changing project source.

## Headless Parity

The TUI and headless commands call the same inventory, intent, planning,
execution, conformance, and reporting kernels. The TUI is the primary
interactive experience; headless commands remain automation and acceptance
surfaces.

The TUI must not gain an untyped mutation path that headless conformance cannot
represent. A capability first introduced conversationally still requires the
same Book and schema contracts.

## Cancellation and Recovery

Cancellation preserves useful conversation and collected answers but removes
approval. A failed proposal retains its diagnostics so the user can revise or
retry without restating the request.

Before mutation, failure leaves the project unchanged. During mutation, Hatmax
uses the execution engine's staging and rollback contracts. When safe rollback
is possible, the working tree returns to its prior state. When rollback would
overwrite concurrent work or failure occurs after retained changes, Hatmax
must report every retained surface and require a new plan and approval.

An environmental validation that the applicable capability classifies as
non-blocking may produce `validation_incomplete`. Hatmax must never describe a
failed check as passed.

## Local State and Privacy

Conversation state lives in the operating system's user-state location, not in
the application repository. The state model is defined in
`conversational-hatmax-surface-model.md`.

Hatmax must not persist:

- authentication material or credentials;
- environment variables;
- arbitrary repository contents;
- hidden model reasoning;
- raw unbounded App Server event streams;
- approval as a reusable authority.

The user can reset local conversation state without modifying the project.
Losing all local state does not prevent Hatmax from reconstructing project
facts from source and the compatible Book.

## Observable Outcomes

The TUI presents at least:

- `conversation_response`;
- `clarification_required`;
- `unsupported`;
- `plan_ready`;
- `cancelled`;
- `plan_stale`;
- `execution_failed`;
- `validation_incomplete`;
- `completed`.

Every outcome distinguishes conversational state, project mutation state,
validation evidence, and the next safe action.

## Acceptance Criteria

- `hm` opens a conversational Hatmax TUI and also exposes headless operations.
- The compatible resident Codex daemon and isolated context are reused across
  turns.
- Ordinary conversation produces no implicit project change.
- Natural language can enter a visible candidate-change state without a
  special command.
- Only typed Book-supported intents become plans.
- Missing required information produces focused adaptive clarification rather
  than a fixed wizard.
- Optional enrichment never delays an already plannable request.
- Every mutation has a visible plan and explicit approval.
- Routine changes can use a compact plan presentation without omitting plan
  content.
- Reopening a project resumes local conversation but reinspects source and
  invalidates stale plans.
- Cancellation and failures preserve useful conversational context without
  preserving approval.
- Headless and TUI operations use the same deterministic kernels.
- Off-domain coding requests cannot turn Hatmax into a general coding harness.

## Open Questions

- Which terminal UI framework and component model best fit the state machine?
- What exact command resets or selects a conversation?
- How long should the `hatmax` compatibility alias remain?
- What retention limit applies to old turns and execution summaries?
- Does prototype testing confirm the compact versus expanded plan presentation?
- Which additional Hatmax-native maintenance capability should follow the
  current operation set first?

<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Hatmax Conversational Application Generation Quiz

Status: Curated

## Objective

Define enough product behavior to specify two related Hatmax generator
capabilities:

- canonical creation of a new Hatmax application;
- a conversational TUI that mediates access to the Hatmax generator and its
  resident Codex interpretation backend.

The quiz covers product behavior only. It does not authorize a specification,
delivery plan, framework selection, or implementation.

## Session State

- Question count: 16
- Remaining budget: 34
- Active topic: complete
- Nesting level: none
- Coverage: scaffold scope 6, adaptive discovery 1, conversational surface 3,
  approval 2, session context 1, failures and recovery 2

## Decisions

### Composite application creation

A request may create only a general Hatmax application or may also provide a
chosen name, a business niche, and one or more initial feature skeletons. The
generator may represent a detailed request as a composite plan with distinct
application and feature units.

### Adaptive discovery

The generator must accept both terse prompts and near-specification prompts.
It must not use a fixed wizard, a fixed question count, or a mandatory sequence
of questions. It accumulates relevant decisions from the conversation and asks
only for information that is still necessary.

### Sufficiency threshold

The product distinguishes information that is required for a valid plan from
information that only enriches the result. It must not proceed while required
information is missing. Once the request is sufficient to produce a useful
Hatmax application, it should stop asking enrichment questions and present the
plan. Later commands can grow the application.

### Authority boundary

Reaching the sufficiency threshold authorizes plan production, not project
mutation. The existing explicit approval boundary remains in force.

### Session directory as the target

The directory in which the Hatmax session was opened is normally the base
directory for application creation. The generator derives a new child
directory from the project slug. For example, a `Real Estate` application
created from a projects directory targets its `real-estate` child. The user
does not need to create or enter that child directory first.

The base location is therefore session context rather than a conversational
question. The generator must still establish that the derived target is safe
and admissible before proposing creation.

### Application identity

The application name is required. Related technical identifiers should be
derived when possible instead of requiring redundant answers. Normalization
must account for the different constraints of a human-readable application
name, a repository or directory slug, a Go module path, and Go package
identifiers. A description may enrich the application but is not required.

The module path uses this precedence:

1. derive it from an existing Git remote when the result is unambiguous;
2. derive it from an explicitly configured module-base preference when one is
   available;
3. ask for the module path when neither source exists.

The generator must not invent a hosting service or owner. A user-level module
base, such as a value stored under the normal configuration directory, is a
possible later convenience rather than a first-version requirement.

### Target directory admission

Application creation applies compatibility and collision checks to the
derived child target instead of requiring the user to prepare it first:

- an empty directory is admissible;
- Git metadata and non-conflicting repository files are preserved and listed
  in the plan;
- an existing compatible Hatmax application selects evolution mode instead of
  `create_application`;
- another Go module or an incompatible application blocks creation;
- a scaffold path collision blocks creation unless a canonical Hatmax merge
  rule explicitly admits it.

### Minimum useful scaffold

A general application scaffold is neutral but runnable. It includes the
canonical Hatmax composition and lifecycle, configuration, logging, web and
Postgres wiring, migration support, a neutral page, basic tests, and standard
project commands. It does not invent domain features or generate unrequested
Diataxis documentation.

Successful compilation is the mandatory scaffold acceptance condition.
Generated tests are desirable, but their execution is a secondary,
best-effort validation because it may depend on external infrastructure. Test
execution that is unavailable must not make an otherwise compiling scaffold
unusable.

An observed test failure caused by generated behavior is a generation failure.
A test that is not run or is blocked by unavailable infrastructure is
non-blocking and must be reported explicitly.

### Conversational mutation boundary

The TUI detects Hatmax mutation intent from ordinary language. It does not
require a command such as `/change`. Ordinary conversation can remain broad
and informal, but only an intent admitted by the Hatmax Book can enter the
proposal, planning, approval, and execution lifecycle. Conversation alone
never creates mutation authority.

### Product scope beyond initial generation

The product may perform Hatmax-native maintenance work beyond creating an
application or feature. Testing and fixture work are candidate capabilities
when they remain governed by Hatmax's architecture and conventions. The term
`generator` may be too narrow if this broader application lifecycle is part of
the intended product identity. Individual maintenance examples are
illustrative, not required capabilities or an exhaustive product boundary.

### Product identity

The user-facing product is Hatmax. Its conversational surface is described as
a conversational Hatmax application builder. `Generator engine` remains a
valid internal component name, but neither `agent` nor `generator` defines the
entire user experience. The canonical short command is `hm` for both the TUI
and headless operations. The existing `hatmax` command remains temporarily as
a compatibility alias. Command-name availability must be verified before
implementation.

### Plan presentation

Every mutation retains a visible plan and explicit approval. Presentation is
provisionally proportional to scope: a routine single operation uses a compact
inline plan, while application bootstrap and other composite work use an
expanded multi-unit plan. This is a usability default to validate with the
first TUI prototype, not a reason to delay the specification.

### Persistent project conversation

Hatmax resumes the prior local conversation when `hm` is reopened for the same
project. This avoids requiring the user to restate relevant context. On every
resume, Hatmax reinspects the project source. Source remains authoritative and
any pending plan invalidated by project drift becomes stale and cannot be
approved or executed.

### Cancellation and recovery

Cancellation preserves the conversation but grants no continuing approval.
Failed work retains the proposal, collected answers, and diagnostics so the
user can revise or retry without starting over. The working tree remains
unchanged when staging and safe rollback make that possible. If a failure
occurs after changes can no longer be rolled back safely, Hatmax must identify
every retained change and require a new approval before further mutation.

## Sufficiency

The quiz provides sufficient pre-specification context for separate drafts of
the canonical application scaffold and conversational product surface. The
remaining questions are bounded design details that can stay explicit in
those drafts and do not require further discovery before drafting.

## Tensions

- More questions can improve the initial result but can also make ordinary
  application creation burdensome.
- Flexible interpretation is desirable, but the definition of a plannable
  Hatmax application must remain deterministic.
- A detailed prompt can describe several operations, but each operation must
  remain visible and reviewable inside the resulting plan.
- A mandatory plan must not impose the same presentation weight on a routine
  feature change and a multi-unit application bootstrap.

## Working Assumptions

- `create_application` will be a first-class typed intent.
- Initial features can be separate units inside one composite proposal.
- Codex may choose natural clarification wording, while deterministic Hatmax
  validation decides whether required information is missing.
- The current session directory is the default parent of the creation target.

## Related Context

- `ops/default/spec/interactive-hatmax-generator.md`
- `ops/default/spec/interactive-product-surface.md`
- `ops/default/ticket/open/20260929085205-explore-conversational-generator-tui.md`
- `ops/default/spec/canonical-application-scaffold.md`
- `ops/default/spec/conversational-hatmax-surface.md`
- `ops/default/spec/conversational-hatmax-surface-model.md`

## Open Questions

- Where should optional generator-wide identity defaults be configured?
- Which Hatmax-native maintenance operations belong inside the product?
- How may a user start a fresh conversation without deleting project state?
- Where is local conversation state stored and how is its lifecycle managed?
- Does the first TUI prototype confirm the provisional adaptive plan
  presentation?

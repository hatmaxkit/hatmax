<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Raw Question and Answer Capture

## Question 1

When a user asks to create an application, should the first approvable
operation create only the runnable scaffold, infer an initial feature, or
present separate `create_application` and `create_feature` units?

### Answer

The interaction must accept an arbitrary amount of initial detail. A user may
request a general Hatmax application, identify a business niche, choose its
name, and request one or more initial feature skeletons. The generator should
collect the information needed to begin useful work and ask follow-up
questions when the request is not sufficient. This must not become a fixed or
fully choreographed question loop.

## Question 2

When the intent is sufficient but could still be enriched, should the TUI stop
asking and present a plan with visible defaults and inferences?

### Answer

Yes. Continuing to ask optional enrichment questions would make the agent
burdensome. Later commands exist to grow the application. The agent should
still investigate when the available information is not sufficient to
proceed.

## Question 3

Is the irreducible input for `create_application` limited to a safe target,
Go module identity, and an application name, while Hatmax, Postgres, and the
server-rendered architecture remain canonical defaults?

### Answer

Yes, with the preference that the generator operate in the directory where
the session was opened instead of asking for another target. The application
name is important. The module or package identity should be derived when it
does not coincide with that name. The generator should normalize natural
names such as `Real Estate`, while respecting that hyphens are not idiomatic
in Go package names. A description can be collected but is not mandatory.

## Question 4

Should the module path be derived from an existing Git remote, with a question
only when the repository context does not determine it?

### Answer

Yes. The generator should attempt derivation when a remote exists. A generic
user configuration could optionally define a default module base, such as a
hosting service and owner prefix to which the project segment is appended.
This may be useful, but it could be excessive for the initial scope and should
not distract from the core interaction.

## Question 5

Should target admission permit an empty directory or a Git repository with
non-conflicting files, select evolution mode for an existing Hatmax
application, and block incompatible modules or scaffold collisions?

### Answer

Yes. The compatibility- and collision-based admission policy is accepted.
The session directory is the base location, not necessarily the application
directory itself. For example, starting a session in a projects directory and
creating `Real Estate` should create its `real-estate` child directory. The
user should not need to create or enter that child directory first.

## Question 6

Should a minimal general scaffold be a neutral but runnable application with a
small composition root, Hatmax lifecycle wiring, configuration, logging,
Postgres, a base migration mechanism, a neutral page, basic tests, and
canonical project commands, but no invented domain feature or unrequested
Diataxis documentation?

### Answer

Yes. The scaffold must compile. It should ideally include tests that are not
red, but successful compilation is the most important condition.

## Question 7

If compilation succeeds but tests cannot run because an external prerequisite
such as Postgres is unavailable, should the generated project be retained with
an explicit incomplete-validation result, while compilation failures and
actual generated-test failures remain unsuccessful outcomes?

### Answer

The testing requirement should be relaxed. Compilation is the important
scaffold gate; test execution can depend on external infrastructure and should
not become an unnecessarily strict blocker.

### Follow-up (Question 8)

Should an observed failing test remain a generation failure, as distinct from
a test that could not run?

### Answer

Accepted default: a real test failure blocks generation. A test that is not
run or is blocked by unavailable infrastructure is non-blocking and reported.

## Question 9

Should the TUI detect Hatmax mutation intent from ordinary language and make
the transition into a visible proposal state, without requiring a special
command such as `/change`?

### Answer

Yes. The experience should not feel materially different from a more general
conversational harness while the user remains within Hatmax application work.
Conversation may cover anything, but not everything becomes a project
mutation. The product must not become a general-purpose coding harness.

The product may also perform testing, fixture, and other work aligned with the
Hatmax core. This raises whether calling the product only a generator is too
limiting.

### Follow-up (Question 10)

What does `CRAP` refer to in the proposed set of Hatmax-native tasks?

### Answer

`CRAP testing` refers to Change Risk Anti-Patterns, which combines code
complexity and test coverage. It was only a spontaneous example and does not
need to appear in the specification.

## Question 11

Should the user-facing product retain the narrow `generator` name, adopt a
broader agent or workbench name, or simply be Hatmax with a conversational
application-building surface?

### Answer

The recommendation is accepted: the product is Hatmax and its user-facing
surface is a conversational Hatmax application builder. The command can still
use the shorter name `hm`.

### Follow-up (Question 12)

Should `hm` become the canonical command while the existing `hatmax` command
is retained for compatibility, or should the two commands have distinct
roles?

### Answer

`hm` should be the canonical command because short two-letter commands are
preferred and no known collision is expected. It should serve both the TUI and
headless operations, with `hatmax` retained temporarily for compatibility.

## Question 13

Should a composite proposal be revised conversationally and then approved once
as one atomic plan, without prompting separately for each application or
feature unit during execution?

### Answer

Atomic approval may improve consistency for initial application generation
because the agent can reason with the complete requested context. It must not
make routine feature creation ceremonial. The distinction may be appropriate
only for a composite initial-generation request.

### Follow-up (Question 14)

Should plan presentation adapt to scope: a compact inline plan and one approval
for a routine feature, but an expanded multi-unit plan for an application
bootstrap or other broad change?

### Answer

There is no strong preference yet, and the uncertainty should not block
progress. Use the recommended adaptive presentation provisionally and validate
it through the initial TUI experience.

## Question 15

Should reopening `hm` in a project resume its prior conversation and pending
context by default, while re-inspecting the source and invalidating stale
plans?

### Answer

Yes. Resuming project conversation should be practical and prevents repeated
explanation. Hatmax must still re-inspect the project rather than treating
conversation as authoritative.

## Question 16

Should cancellation or a failed admitted mutation leave the working tree
unchanged while retaining the proposal and diagnostics in the conversation so
the user can revise or retry it?

### Answer

Yes, when technically possible. The conversation and diagnostics should remain
available for correction or retry. The specification must not claim that the
working tree is unchanged when safe rollback is no longer possible.

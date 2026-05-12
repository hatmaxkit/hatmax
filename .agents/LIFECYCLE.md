# Delivery Lifecycle Contract

This document defines lifecycle rules for executing work in this repository.

## 1. Work phases

1. Alignment
2. Planning (only when requested)
3. Implementation
4. Validation
5. Report and handoff

## 2. Entry gates

- Alignment starts when task context is provided.
- Planning starts only when explicitly requested.
- Implementation starts only after task/plan approval.

## 3. Exit gates

A work cycle is complete only when:
- requested scope is implemented
- relevant checks are reported
- current status and remaining risks are explicit

## 4. Blocked state handling

If blocked:
- report blocker precisely
- include command/error context
- stop without speculative side changes

## 5. Handoff artifact policy

- `ops/handovers/` is runtime scratch space and is not versioned.
- Durable delivery records live in `ops/plans/*`, `ops/trackers/*`, and `ops/artifacts/*` when the project uses the ops structure.

# Work Request Routing

This document defines how incoming requests are classified for execution workflow.

## Objective

Route requests into one of these execution modes:
- discussion
- planning
- implementation
- investigation
- maintenance/meta

## Rules

- Questions are treated as discussion unless implementation is explicitly requested.
- Planning is produced only when requested.
- Implementation starts only with explicit instruction.
- Ambiguous requests should be clarified before code changes.

## Non-goal

This document does not define product/runtime behavior.

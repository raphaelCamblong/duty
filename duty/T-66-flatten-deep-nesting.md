---
id: T-66
title: Flatten deep nesting
status: todo
blocked-by: []
---

# T-66 — Flatten deep nesting

## Goal
No function in internal/ or cmd/ runs three levels of indentation deep —
nested loops and loop-in-if-in-loop bodies get extracted into named helpers.

## Read first
CLAUDE.md's new shallow-indentation rule; the funlen-35 constraint the
extractions must respect.

## Scope
- Scan for functions whose bodies nest 3+ levels (loops in loops in
  conditionals count; switch cases count as one level). For each: extract
  the inner unit into a named helper whose name states what the level does —
  the same move the comment rule prescribes.
- Judgment allowed: a 3-level spot where extraction genuinely obscures (a
  tight 4-line double loop, e.g. matrix-ish walks) may stay, justified in
  the report.
- Behavior frozen, zero test edits, lint stays at 0 issues.

## Out of scope
Test files; logic changes; the task-representation redesign (separate task —
don't pre-empt it with structural changes to list/get/scan beyond extraction).

## Gates
- [ ] Post-scan shows no unjustified 3+-level function in internal/ + cmd/.
- [ ] just check green with zero test edits.

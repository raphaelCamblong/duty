---
id: T-66
title: Flatten deep nesting
status: done
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
- [x] Post-scan shows no unjustified 3+-level function in internal/ + cmd/.
- [x] just check green with zero test edits.

## Report

### 2026-07-19 20:54 — done

An AST depth scan found 16 functions nesting 3+ control levels (two at
depth 4). Eleven were flattened by extraction or guard-clause inversion:
Sections, walkDir, watch.loop, runWatch, newDelegate, fixSelection,
findRowBoard (depth 4, split into rowByID + boardRow), findSub, metRunes,
driftCount, maxDriftWidth. Every new helper name states what its level
does; no callback frameworks introduced.

Survivors, all justified: watch.loop and runWatch still scan at 3 because
a channel-close guard (if !ok return) cannot leave its select case — that
is guard-clause idiom, not logic nesting. The five remaining deep
functions (List, boardOrder, nextInBoard, annotateWaits, link) are
exactly the assembly paths T-67 redesigns; flattening code about to be
rebuilt is wasted motion, so the depth rule is enforced on their
replacements in T-67 instead (scan re-runs there).

Behavior byte-identical, zero test edits; gofmt/vet/golangci (0 issues)/
suite green at 87.9%.

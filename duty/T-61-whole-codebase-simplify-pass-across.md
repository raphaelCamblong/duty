---
id: T-61
title: Whole-codebase simplify pass across internal/
status: todo
blocked-by: []
---

# T-61 — Whole-codebase simplify pass across internal/

## Goal
Every package under `internal/` gets a real /simplify pass — reuse,
simplification, efficiency, altitude — not just the diff-scoped cleanups
individual tasks have carried so far.

## Read first
CLAUDE.md — the full code rules (interfaces at the consumer only, errors
wrapped, quiet on success, naming, godoc-one-line, NO narrating inline
comments, small functions/guard clauses, no package-level state, line-surgical
edits, dependency rule, funlen). That's the entire bar. Do NOT read past task
reports or archived task files — judge the code as it stands today, fresh,
with no assumption that a prior pass already covered a file well.

## Scope
- One /simplify-style pass per package (or logical group — task+board
  together, cli+app together, tui alone given its size), four angles: reuse,
  simplification, efficiency, altitude. Independent reviewers per angle,
  read the code fresh, then dedup and apply.
- Also actively hunt, across every file: comments that violate the rule
  (anything narrating what code already says — delete or replace with an
  extracted, well-named function; keep only comments stating a constraint
  the code can't express) and any drift from the code rules above
  (interfaces with one implementation, functions creeping past 35 lines,
  naming stutter, missed error wrapping, anything not quiet-on-success).
- Judge design and implementation on merit: is this the best shape for what
  it does, not just "does it pass lint". Better data structures, clearer
  control flow, dead abstractions, real optimization opportunities — flag
  and fix what's genuinely better, not cosmetic churn.
- Behavior frozen throughout: the full test suite is the referee, zero
  existing-test edits unless a finding is explicitly sanctioned in the same
  change (document any such case).
- Fix what's genuinely worth it; skip with one-line reasons what isn't —
  quality pass, not a rewrite. No new features, no architecture changes
  without flagging them separately instead of just doing them.

## Out of scope
Behavior changes; new features; test refactoring; anything touching
`docs/*.md` content (code only).

## Gates
- [ ] Every package under internal/ has a documented pass (findings applied
  or explicitly cleared) in the final report, based on reading the current
  code — not prior task history.
- [ ] A dedicated comment sweep: every remaining comment in internal/ is
  either an exported-symbol godoc (one line) or a genuine constraint the
  code cannot express — none narrate what the code does.
- [ ] `just check` green throughout (funlen 35, golangci-lint 0 issues);
  zero test edits beyond sanctioned, documented exceptions.
- [ ] Report totals: findings found / applied / skipped, packages already
  clean, and comments removed.

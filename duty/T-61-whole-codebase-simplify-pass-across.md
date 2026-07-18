---
id: T-61
title: Whole-codebase simplify pass across internal/
status: backlog
blocked-by: []
---

# T-61 — Whole-codebase simplify pass across internal/

## Goal
Every package under `internal/` gets a real /simplify pass — reuse,
simplification, efficiency, altitude — not just the diff-scoped cleanups
individual tasks have carried so far.

## Read first
CLAUDE.md (architecture + code rules — the bar every finding is judged
against); the accumulated per-task simplify addenda in `duty/archive/` (skip
what was already reviewed recently: T-28, T-38-T-40 palette, T-50/T-52
visibility round, T-58/T-57 backlog+archive — focus fresh eyes on packages
that predate those passes or were never reviewed as a whole: task, board,
tree, fsys, config, humanize, fetch, watch, and a full app/cli/tui sweep
rather than the incremental diffs they've had).

## Scope
- One /simplify-style pass per package (or logical group — task+board
  together, cli+app together, tui alone given its size), four angles:
  reuse, simplification, efficiency, altitude. Use independent reviewers
  per angle, then dedup and apply.
- Behavior frozen throughout: the full test suite is the referee, zero
  existing-test edits unless a finding is explicitly sanctioned in the
  same change (document any such case).
- funlen 35 / golangci-lint 0 issues maintained (already the bar — this
  pass should find it mostly clean; report what's genuinely improved vs.
  what's already good).
- Fix what's genuinely worth it; skip with one-line reasons what isn't —
  this is a quality pass, not a rewrite. No new features, no architecture
  changes without flagging them separately instead of just doing them.

## Out of scope
Behavior changes; new features; test refactoring; anything touching
`docs/*.md` content (code only).

## Gates
- [ ] Every package listed has a documented pass (findings applied or
  explicitly cleared) in the final report.
- [ ] `just check` green throughout; zero test edits beyond sanctioned,
  documented exceptions.
- [ ] Report totals: findings found / applied / skipped, and which
  packages were already clean (a useful signal on its own).

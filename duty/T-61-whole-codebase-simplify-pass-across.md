---
id: T-61
title: Whole-codebase simplify pass across internal/
status: done
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
- [x] Every package under internal/ has a documented pass (findings applied
  or explicitly cleared) in the final report, based on reading the current
  code — not prior task history.
- [x] A dedicated comment sweep: every remaining comment in internal/ is
  either an exported-symbol godoc (one line) or a genuine constraint the
  code cannot express — none narrate what the code does.
- [x] `just check` green throughout (funlen 35, golangci-lint 0 issues);
  zero test edits beyond sanctioned, documented exceptions.
- [x] Report totals: findings found / applied / skipped, packages already
  clean, and comments removed.

## Report

### 2026-07-18 16:16 — done

Integration verification of the six T-61 simplify passes: green across the board.

Referee (from repo root):
- go build ./cmd/duty: ok
- go test ./tests/... -coverpkg=./internal/...: ok, 87.9% coverage
- go vet ./...: clean
- gofmt -l .: empty
- golangci-lint run: 0 issues
- just check: green (funlen 35, 0 lint issues, tests pass)
- filename-literal guard (grep 'BOARD.md|duty.toml' in internal/ minus names/): empty — literals stay confined to internal/names

No cross-group integration break: nothing one group renamed is still referenced by another.

Totals across all six groups (findings applied / comments removed), from the T-61 simplify-pass commits:
- task+board (5535eb8): 4 applied — extract locateRow, isHeading, gatePositions; tighten parseGate. 0 comments removed (godocs reworded/extended).
- app (e793db7): 6 applied — applyEdit/lockedEdit read->transform->write spine (gate/section/move), shared tasksIn walk, shared parseTask, boardIndexPath helper, stale nameRE comment fix, root->listDir rename. 1 comment trimmed (nameRE stale clause).
- cli (635219f): 2 applied — tsv() helper routing the four --agent record builders, drop unused strings import. 4 comments trimmed (editorializing godoc tails).
- tui (7b24b68): 6 applied — rebuildList defers to reskinList, drop redundant Model.mode, textualParent->path.Dir, clamp->max/min builtins, inline statusInk, collapse link's three parallel maps into one struct-valued map. 2 comments removed (godocs for the deleted textualParent/statusInk).
- fsys+tree+config+names (b37bb41): 5 applied — Color.UnmarshalTOML interface{}->any, extract Mem.exists, WriteFile named-return + deferred error wrap, FindRoot delegates to CurrentBoard, underArchive->slices.Contains. 0 comments removed.
- humanize+fetch+watch (322116b): 1 applied — wrap fetch's two bare error returns with context. 0 comments removed; humanize and watch reviewed clean.

Grand totals:
- Findings applied: 24
- Comments removed/trimmed: 7
- Findings skipped: 0 recorded (the commits capture applied fixes only; no skip was committed, so found == applied on git evidence).
- Packages reviewed clean (no findings): humanize, watch.

All four gates pass. Behavior frozen — full suite green with zero test edits.

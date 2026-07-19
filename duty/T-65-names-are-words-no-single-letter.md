---
id: T-65
title: "Names are words: no single-letter identifiers"
status: in-progress
claimed-by: fable
blocked-by: []
---

# T-65 — Names are words: no single-letter identifiers

## Goal
Every parameter and variable across internal/ and cmd/ carries a real name —
`scope` not `s`, `index` not `x` — per the new CLAUDE.md rule.

## Read first
CLAUDE.md's naming rule and its two sanctioned exceptions: conventional short
receivers (`a App`, `m Model` — Go idiom, CLAUDE.md mandates short receivers)
and `i`/`j` loop indices in loops a few lines long.

## Scope
- Sweep every .go file under internal/ and cmd/: rename single-letter
  parameters and variables to full words matching their type or role
  (`s Scope` → `scope Scope`, `f fsys.FS` → `fs`, `b Board` → `board` unless
  it shadows the package name — then a role word like `current` or
  `boardView`). Package-name shadowing is the one trap: never rename a
  variable to collide with an imported package identifier used in the same
  function.
- Receivers stay short (documented judgment call — flag any receiver that
  feels ambiguous for a follow-up rather than renaming unilaterally).
- Pure rename: zero signature *shape* changes, zero behavior changes, zero
  test edits (parameter names are invisible to callers).

## Out of scope
Receivers; test files; exported symbol renames; anything beyond identifier
names.

## Gates
- [ ] An AST scan for single-letter params/vars (excluding receivers, blank,
  and short-loop indices) reports zero offenders across internal/ + cmd/,
  or each survivor is justified in the report.
- [ ] just check green with zero test edits.

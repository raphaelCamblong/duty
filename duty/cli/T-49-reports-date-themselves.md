---
id: T-49
title: Reports date themselves
status: todo
blocked-by: []
---

# T-49 — Reports date themselves

## Goal
Every `duty report` block opens with a dated heading the writer didn't have
to remember — the task file becomes a self-dating log.

## Read first
`internal/app/report.go` + `task.AppendReport`; CLAUDE.md (no package-level
state — the clock must be injectable); how agents hand-wrote `### 2026-07-09
— done` headers inconsistently.

## Scope
- `duty report` prefixes each appended block with `### 2006-01-02 15:04`
  (local time), plus ` — <status>` when `--status` is given. Blank line
  after the heading, then the stdin content verbatim.
- Clock injection: `App` gains a `now func() time.Time` (constructor default
  `time.Now`; tests inject a fixed clock — the seam exists for the test
  double, per the interface rule).
- Existing report content is never touched; only new appends gain headings.
- Update the tests that pin report output bytes (sanctioned — the new bytes
  include the fixed test clock); docs/tasks.md report paragraph mentions the
  automatic dating.

## Out of scope
Retro-dating old reports; timezones/config keys; dating `set` edits.

## Gates
- [ ] Two reports appended in a scratch tree produce two dated headings,
  content verbatim beneath, older blocks untouched (byte test with a fixed
  clock).
- [ ] `report --status done` heading reads `### <stamp> — done`.
- [ ] `just check` green; docs updated; no package-level clock state.

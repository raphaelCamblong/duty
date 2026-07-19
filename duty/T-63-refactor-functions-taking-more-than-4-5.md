---
id: T-63
title: Refactor functions taking more than 4-5 parameters
status: backlog
blocked-by: []
---

# T-63 — Refactor functions taking more than 4-5 parameters

## Goal
The 31 functions currently taking more than 4 parameters get real parameter
objects where it helps readability — the AST-scan list below is the starting
point, not a guess.

## Read first
CLAUDE.md's new rule (>4 params is a smell — bundle into a struct). Full
scan (internal/, >4 params, most-to-least):
newDelegate(8): internal/tui/entry.go:163 [Theme,*zone.Manager,Board,bool,bool,bool,time.Time,string]
statusWrite(8): internal/app/status.go:49
createTaskLocked(7)/writeTask(7)/CreateTask(7): internal/app/create.go
getTaskOut(7): internal/cli/get.go:92
InstallSkill(6): internal/app/skill.go:70
runWatch(6): internal/cli/watch.go:53
newSkillInstallCmd(6): internal/cli/skill.go:51
moveAcross(6): internal/app/move.go:124
AddRow(6): internal/board/board.go:89
Report(6): internal/app/report.go:21
plus ~19 more at 5 params across app/move.go, app/status.go, app/section.go,
cli/cli.go, cli/get.go, cli/skill.go, app/list.go, app/get.go, tui/entry.go,
tui/scan.go — full list in this task's Report once re-run.

## Scope
- Per offending function (or tight cluster sharing the same params, e.g.
  Move/moveTrack/relocate/reorder in move.go, or SetStatus/setStatusLocked/
  guardClaim in status.go): introduce ONE parameter struct capturing the
  related fields, or split into 2 structs when params group naturally
  (e.g. "where" + "what"). Judgment call per site — do not mechanically
  wrap every param into one giant struct.
- Constructors/call sites updated accordingly; unexported params stay
  unexported fields.
- Zero behavior change — existing tests are the referee, near-zero test
  edits expected (call sites inside tests may need mechanical updates if
  they construct these calls directly; list every one touched).
- Re-run the AST param-count scan at the end and include the after-list in
  the report (some entries may legitimately remain at 5 if a struct split
  doesn't clarify anything — judgment, not a mechanical rule).

## Out of scope
Comment pruning (separate task); new features; changing what any function
does, only how its inputs are grouped.

## Gates
- [ ] Every function above 6 params refactored to a parameter struct (or
  judged and explicitly left, with why, in the report).
- [ ] `just check` green; full suite green with only mechanical test-call
  updates (listed).
- [ ] Before/after AST scan counts in the report.

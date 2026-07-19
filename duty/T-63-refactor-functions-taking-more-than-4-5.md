---
id: T-63
title: Refactor functions taking more than 4-5 parameters
status: done
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
- [x] Every function above 6 params refactored to a parameter struct (or
  judged and explicitly left, with why, in the report).
- [x] `just check` green; full suite green with only mechanical test-call
  updates (listed).
- [x] Before/after AST scan counts in the report.

## Report

### 2026-07-19 19:28 — done

Bundled the many-parameter functions into parameter structs. Re-scanned the
AST first (T-62's comment prune had shifted line numbers) — the before list was
31 functions with >4 params. After: 7, all at exactly 5 and each judged worth
leaving.

## Before (AST scan, >4 params)
8: statusWrite, newDelegate
7: CreateTask, createTaskLocked, writeTask, getTaskOut
6: moveAcross, Report, InstallSkill, AddRow, newSkillInstallCmd, runWatch
5: buildTaskInfo, taskRow, reorder, Move, relocate, moveTrack, editSection,
   installClaude, SetStatus, setStatusLocked, guardClaim, newRoot, Run,
   newGetTasksCmd, newSkillCmd, trackLine, archivedLine, taskLine, joinRow
TOTAL: 31

## After (AST scan, >4 params)
5: buildTaskInfo, taskRow, reorder, editSection, newRoot, Run, newGetTasksCmd
TOTAL: 7  (0 above 5)

## Structs introduced
- app.StatusChange {ID, Status, As, Force} — SetStatus, setStatusLocked,
  statusWrite, guardClaim, Report all thread it (exported: cli + tests build it).
- app.TaskSpec {Title, Slug, Section, BlockedBy} — CreateTask, createTaskLocked,
  writeTask.
- app.Dest {Track, Section} — Move, relocate, moveTrack (paired with the
  existing Position for the reorder phase).
- app.Install {Target, Cwd, Home, User, Force} — InstallSkill, installClaude.
- app.across (unexported) — moveAcross's one cross-board-move bundle.
- board.AddRow now takes the existing board.Row {ID, File, Title, Status} plus
  section, instead of five loose strings.
- tui.viewOpts {showAge, showGates, showArchive, now, glyph} — newDelegate.
- tui.matches {head, tail} — splitMatches returns it; trackLine/archivedLine/
  taskLine take it.
- tui.boardFiles {files, bad, used} — joinRow and appendOrphans.
- cli.taskQuery {id, section, body, agent} — getTaskOut.
- cli.skillCtx {app, fetcher, cwd, home, out} — newSkillCmd, newSkillInstallCmd.
- cli.watchCmd {app, fs, cwd, out} — runWatch.

24 functions refactored; behaviour frozen (the round-trip and full suite are
byte-for-byte unchanged).

## Left at 5 (judged, no clarifying split)
- buildTaskInfo / taskRow — TaskInfo/Row constructors; their 5 inputs (root or
  board context + file location + parsed content + mtime) are all distinct, no
  subset is a value object. A wrapper would just relist the fields.
- reorder — root (for ref resolution) + board bytes + path + filename + Position
  are unrelated; Position is already a struct.
- editSection — cwd, id, kind, reader, edit-callback: the shared spine of
  SetSection/SetSections, all distinct.
- Run / newRoot — the conventional CLI entry shape (args + stdin/stdout/stderr +
  version); a streams struct would obscure a familiar signature and churn the
  public entry + every cli.Run test for no readability gain.
- newGetTasksCmd — the (use, hidden) pair just parameterises two registrations
  (tasks vs list); wrapping two scalars adds a type without clarifying.

## Gates
- go build -o bin/duty ./cmd/duty: ok
- go test ./tests/... -coverpkg=./internal/... -count=1: ok (87.9%)
- golangci-lint run: 0 issues
- gofumpt -l . / gofmt -l .: empty
- go vet ./...: clean
- just check: green

## Mechanically-touched test files (call syntax only, no assertion weakened)
- tests/board_test.go — 2 board.AddRow calls now pass board.Row{...}
- tests/cli_mutate_test.go — 2 a.Report calls now pass app.StatusChange{...}
- tests/cli_skill_test.go — 1 a.InstallSkill call now passes app.Install{...}

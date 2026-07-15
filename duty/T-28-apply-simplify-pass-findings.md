---
id: T-28
title: Apply simplify-pass findings
status: done
blocked-by: []
---

# T-28 — Apply simplify-pass findings

## Goal
The verified findings from the four-angle /simplify review (reuse, simplification,
efficiency, altitude) are applied: less duplication, single-source policies, no
per-frame waste — observable behavior unchanged except two sanctioned tightenings.

## Read first
`CLAUDE.md`; the fix list below is the full pre-made scope (file:line references
verified by the reviewers on the current HEAD).

## Scope
**Reuse / dedup (app):**
- `app/app.go`: generalize `resolveOpen` → `resolveOpenWithRoot(cwd, id) (root,
  path string, err error)`; `resolveOpen` delegates. Collapse the inlined
  FindRoot+ResolveTask pairs in `GetTask` (get.go:43) and `moveTrack` (move.go:39).
- `app/delete.go:20-27`: use `a.readTask`; `:32-41`: use `a.dropFromBoard`.
- New `App.walkBoards(cwd) (boardDir string, boards []string, err error)` for the
  CurrentBoard→Boards prelude in `List`, `GetTracks`, `GetNext`, `Archive`.
- New unexported `boardBeside(taskPath) string` for the 5×
  `filepath.Join(filepath.Dir(taskPath), names.BoardFile)` (status.go:33,
  delete.go, move.go:57,91,118).

**Shared helpers (right altitude):**
- `tree.TaskFileNames(f fsys.FS, dir string) ([]string, error)` — the
  ReadDir+IsDir/IsTaskFile filter written in 5 places (archive.go:94,116,
  get.go:246, list.go:65, tui/scan.go:155); preserve current ReadDir ordering.
- `board.TitleOr(content []byte, fallback string) string` — the H1-else-basename
  policy duplicated at get.go:138 and tui/scan.go:123.
- `task` owns the id format: `IDPrefix` const + `FormatID`; use in
  app/create.go:47 and build tree's id regex/glob from it.
- `task.ValidSlug(s string) bool` matching exactly what `Slugify` produces
  (non-empty, ≤40, `[a-z0-9-]`, no leading/trailing hyphen); `CreateTask` uses it
  for `--slug` instead of app's `nameRE` (which stays for track names).
  SANCTIONED TIGHTENING: previously-accepted degenerate slugs (>40, edge
  hyphens) now error.

**Simplification (tui/cli):**
- `tui/model.go`: delete `findRow` (strict subset of `findRowBoard`; caller
  view.go:325 uses `findRowBoard` and drops the board). Replace stringly
  `previewKey`+`splitKey` with two fields `previewKind, previewArg`; delete
  `splitKey` and the unreachable `!ok` branches.
- `tui/view.go`: derive `statusStyle(s)` = `Foreground(statusColor(s))`, delete
  the parallel switch and `yellowStyle`/`redStyle`/`greenStyle`; `trackBar` calls
  `statusStyle`. `focusedBox = headerBox` (identical values). `barData` iterates
  the shared `rollupOrder` — SANCTIONED VISUAL FIX: header bar segment order
  aligns with the inline track bars (they currently disagree on screen).
- `tui/entry.go:156,175`: extract `titleStyle(selected bool)`.
- `task/task.go:118`: extract `ensureBlankLine(b *bytes.Buffer)` for the twice-
  repeated blank-line guard in `AppendReport`.
- `cli/cli.go:127`: share the dispatch scaffolding between `rootCmd` and
  `newGroupCmd` (common RunE builder + field setter; exit-2 contract defined once).

**Efficiency:**
- Preview title: `previewTitle()` (view.go:272) runs `findRowBoard`/`findSub` —
  O(tree) — on every animation frame. Compute once in `renderPreview`
  (model.go:407) into a cached model field alongside the body; `rightPanel`
  reads the field. Recompute on re-scan/resize like the body.

**Conditional (apply ONLY if the full suite passes with zero test edits):**
- `app/move.go:62-68`: same-board `--track` special case papers over
  `moveAcross`'s same-path read/write hazard; route it through the byte-
  preserving `moveRow` (board.MoveRow + prune) path instead. First READ
  tests/cli_mutate_test.go:~418 — if it pins the rebuilt-row behavior, SKIP and
  record why in the report.

**Skipped by triage (record in report, do not do):** header-string per-frame
cache (invalidation complexity > O(width) win); `board.RowStatusFor` merge and
TUI drift-enum refactor (both require test rewrites — tests are out of scope);
`taskInfo` inline (name documents intent); `tree.RelBoard` consolidation (the
two sites deliberately differ on error policy); `unknownStatusErr` prose
(byte-frozen message); errStyle/driftStyle stay separate (semantically distinct).

## Out of scope
Test-file edits of any kind; behavior changes beyond the two sanctioned items;
new features; `future.md`.

## Gates
- [x] Full suite green with ZERO test-file edits
  (`go test ./tests/... -coverpkg=./internal/... -count=1`), including
  roundtrip, frame audit, `TestStartupPerformance`.
- [x] `golangci-lint run` clean (committed config, includes gofumpt);
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [x] `findRow`, `splitKey`, the duplicated switches/guards are GONE (grep);
  header bar and track bars render segments in the same order.
- [x] Report lists every applied finding, the two sanctioned behavior notes,
  the conditional's outcome, and the skip list verbatim.

## Report

Applied every Scope item from the four-angle simplify review. Build, vet, gofmt,
golangci-lint (committed config), and the full suite are all green with ZERO
test-file edits.

APPLIED — reuse/dedup (app):
- app.go: added resolveOpenWithRoot(cwd,id)->(root,path,err); resolveOpen now
  delegates. GetTask and moveTrack use it, collapsing their inlined
  FindRoot+ResolveTask pairs.
- app.go: added walkBoards(cwd)->(boardDir,boards,err) for the CurrentBoard->Boards
  prelude; List, GetTracks, GetNext, Archive now share it.
- app.go: added unexported boardBeside(taskPath) for the 5x
  filepath.Join(filepath.Dir(taskPath), names.BoardFile) (status.go, delete.go,
  move.go x3).
- delete.go: uses a.readTask (drops the inline ReadFile+task.Parse) and
  a.dropFromBoard (drops the inline DropRow+PruneEmptySections); board/names
  imports removed.

APPLIED — shared helpers (altitude):
- tree.TaskFileNames(f,dir)->([]string,err): the ReadDir+IsDir/IsTaskFile filter,
  now used by archive.go (doneTasks, countTaskFiles), get.go (taskStatuses),
  list.go (boardRows), tui/scan.go (readTasks). ReadDir ordering preserved. The
  five call-site ReadDir error strings collapse into one ("read dir %s"); no test
  pinned them.
- board.TitleOr(content,fallback): the H1-else-basename policy, now used at
  get.go trackInfo and tui/scan.go scanBoard.
- task.IDPrefix const + task.FormatID(nn): create.go uses FormatID instead of the
  "T-" literal; tree's taskNN regex is built from regexp.QuoteMeta(task.IDPrefix)
  (tree now imports task, allowed inward by the dependency rule).
- task.ValidSlug(s): non-empty, <=40, [a-z0-9-], no leading/trailing hyphen.
  CreateTask validates --slug with it instead of nameRE (nameRE stays for track
  names in track.go). readTasks in scan.go likewise switched to TaskFileNames.

APPLIED — simplification (tui/cli):
- tui/model.go: deleted findRow (strict subset of findRowBoard); its one caller
  in previewContent now uses findRowBoard and drops the board. Replaced the
  stringly previewKey+splitKey with previewKind/previewArg fields (constants
  previewTask/previewTrack); deleted splitKey and the unreachable !ok branches in
  previewTitle/previewContent/DetailID.
- tui/view.go: statusStyle(s) now derives from Foreground(statusColor(s)); the
  parallel switch and yellowStyle/redStyle/greenStyle are gone. focusedBox =
  headerBox (identical values). trackBar calls statusStyle.
- tui/entry.go: extracted titleStyle(selected bool); trackLine and taskLine share
  it (the two "bold-if-selected" blocks removed).
- task/task.go: extracted ensureBlankLine(*bytes.Buffer) for the twice-repeated
  blank-line guard in AppendReport.
- cli/cli.go: extracted dispatchOnly(cmd, missing) — the shared arbitrary-args +
  silenced-output + RunE (exit-2 contract) scaffolding for rootCmd and
  newGroupCmd, defined once.

APPLIED — efficiency:
- Preview title was recomputed (findRowBoard/findSub, O(tree)) every animation
  frame by View and rightPanel. Now computed once in renderPreview into the
  previewTitleText model field; View and rightPanel read the field (truncation
  stays per-frame, O(width)). Recomputed on re-scan/resize like the body, not on
  scroll ticks.

SANCTIONED BEHAVIOR NOTES (two):
1) Slug tightening: task.ValidSlug replaces nameRE for --slug, so previously
   accepted degenerate slugs (>40 chars, leading/trailing hyphen) now error.
   Behavior change is intentional and per scope.
2) Header-bar visual fix: barData now iterates the shared rollupOrder
   (in-progress, todo, blocked, done) instead of its old todo-first order, so the
   header distribution bar and the inline track bars render segments in the same
   order. Intentional per scope.

CONDITIONAL — APPLIED (full suite passed, zero test edits):
- move.go same-board --track special case (was DropRow+Prune+AddRow, rebuilding
  the row) now routes through the byte-preserving board.MoveRow+PruneEmptySections
  path via a new moveRowInBoard(taskPath,section) helper (also shared by the
  --section-only moveRow). Read tests/cli_mutate_test.go:~418 first: that test
  only asserts one row survives and the file stays put — it does NOT pin the
  rebuilt-row bytes, so the conditional was safe. Full suite green afterward, no
  test edits.

SKIPPED BY TRIAGE (verbatim, not done): header-string per-frame cache
(invalidation complexity > O(width) win); board.RowStatusFor merge and TUI
drift-enum refactor (both require test rewrites — tests are out of scope);
taskInfo inline (name documents intent); tree.RelBoard consolidation (the two
sites deliberately differ on error policy); unknownStatusErr prose (byte-frozen
message); errStyle/driftStyle stay separate (semantically distinct).

GATE TAILS:
- go test ./tests/... -coverpkg=./internal/... -count=1 -> ok, coverage 85.6%.
- golangci-lint run (committed config, gofumpt) -> 0 issues; go vet ./... clean;
  gofmt -l . empty; go build -o bin/duty ./cmd/duty ok.
- grep: findRow, splitKey, previewKey, yellowStyle/redStyle/greenStyle all GONE;
  barData and trackBar both iterate rollupOrder (same segment order).

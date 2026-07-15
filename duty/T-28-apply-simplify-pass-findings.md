---
id: T-28
title: Apply simplify-pass findings
status: todo
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
- [ ] Full suite green with ZERO test-file edits
  (`go test ./tests/... -coverpkg=./internal/... -count=1`), including
  roundtrip, frame audit, `TestStartupPerformance`.
- [ ] `golangci-lint run` clean (committed config, includes gofumpt);
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.
- [ ] `findRow`, `splitKey`, the duplicated switches/guards are GONE (grep);
  header bar and track bars render segments in the same order.
- [ ] Report lists every applied finding, the two sanctioned behavior notes,
  the conditional's outcome, and the skip list verbatim.

## Report

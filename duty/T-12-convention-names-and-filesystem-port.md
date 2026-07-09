---
id: T-12
title: Convention names and filesystem port
status: done
blocked-by: []
---

# T-12 — Convention names and filesystem port

## Goal
Two foundations of the architecture refactor: every convention filename defined in
exactly one leaf package, and every filesystem touch behind an `fsys.FS` port.

## Read first
`CLAUDE.md` (the rewritten Architecture section is the target state), the user's
review notes in `future.md` (context only — never edit/commit that file),
`task-system-spec.md` §9.

## Scope
- `internal/names` (new, zero deps): `BoardFile` ("BOARD.md"), `ConfigFile`
  ("duty.toml"), `ReadmeFile` ("README.md"), `ArchiveDir` ("archive"), `TreeDir`
  ("duty"). Every package — including `board`'s rendered-content strings, now legal
  since `names` is leaf vocabulary — references these; delete the constants from
  `internal/tree` and every duplicated literal repo-wide.
- `internal/fsys` (new): `FS` interface with exactly the operations the system needs
  (read file, atomic write, rename, remove, mkdir-all, read dir, stat, walk); adapter
  `OS` (absorbs `fsutil.WriteAtomic`'s temp+rename logic); adapter `Mem` (minimal
  map-backed in-memory double for fast unit tests).
- Delete `internal/fsutil`; rewire `internal/tree`, `internal/config`, and the current
  `internal/cli` handlers to receive/use `fsys.FS` (constructor or parameter
  injection — no package-level state).
- Behavior is frozen: the entire existing test suite (incl. the T-09 invariants) must
  pass unchanged, except mechanical call-site updates where a signature gained an
  `fsys.FS` parameter.
- Add `tests/fsys_test.go`: `OS` and `Mem` pass one shared behavior table
  (write-then-read, atomic replace, rename, walk order).

## Out of scope
The app service layer and cobra (T-13); comment sweep (T-14); any behavior or file
format change.

## Gates
- [x] Zero convention-filename literals outside `internal/names`
  (`grep -rn 'BOARD\.md\|duty\.toml' internal/ --include='*.go' | grep -v names/`
  is empty).
- [x] Zero `os.` file calls outside `internal/fsys`
  (reads/writes/rename/remove/mkdir/walk all go through the port).
- [x] `go test ./tests/... -coverpkg=./internal/...` green including the invariants
  suite; shared `OS`/`Mem` behavior table green.
- [x] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

Refactor complete: convention names centralized and every filesystem touch put behind an `fsys.FS` port.

Files changed
- New `internal/names` (zero deps): BoardFile, ConfigFile, ReadmeFile, ArchiveDir, TreeDir. Every duplicated filename literal repo-wide now references these; the old constants were removed from `internal/tree`, and comments/error strings/regexps that embedded the literals (board footer regexp, board.Render, AddBoardBullet, init readme, tree's second-config error) were rewired through the constants.
- New `internal/fsys`: FS interface (ReadFile, WriteFile [atomic], Rename, Remove, MkdirAll, ReadDir, Stat, WalkDir) with `OS` (absorbs the old fsutil temp+rename atomic write) and `Mem` (map-backed double, faithful WalkDir honouring SkipDir/SkipAll, ErrNotExist on missing files).
- Deleted `internal/fsutil` and `tests/fsutil_test.go`.
- Rewired `tree`, `config`, all `cli` handlers, and the TUI scan/watch layer to receive/use `fsys.FS` by parameter injection (no package-level state). `cli.Run` constructs `fsys.OS{}` and threads it to handlers; the TUI's directory walk now goes through the port while fsnotify stays. main.go untouched (main->app wiring is T-13).
- New `tests/fsys_test.go`: one shared behavior table both OS and Mem pass (write-then-read+0644, atomic replace with no residue, rename, walk order, missing-file ErrNotExist, missing-dir write error).

Gates (all green)
- `grep -rn 'BOARD\.md\|duty\.toml' internal/ --include='*.go' | grep -v names/` -> empty.
- No `os.*` file calls or `filepath.WalkDir` outside `internal/fsys` (only os.Getwd/Getenv/UserConfigDir remain, which are not file ops).
- `go test ./tests/... -coverpkg=./internal/... -count=1` -> ok, 81.3% (also green under -race); the T-09 invariants suite passes unchanged; OS/Mem table green.
- `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

Deviations / notes
- Behavior frozen: test changes were purely mechanical call-site updates for signatures that gained an `fsys.FS` parameter (tree/config/tui helpers), plus swapping `tree.BoardFile` for `names.BoardFile` in one TUI test helper.
- `config.UserPath`'s user-config filename ("config.toml") is intentionally left as-is: it is the user-config path, distinct from the project ConfigFile ("duty.toml"), and not part of the five-name convention vocabulary.

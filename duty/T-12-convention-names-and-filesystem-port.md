---
id: T-12
title: Convention names and filesystem port
status: todo
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
- [ ] Zero convention-filename literals outside `internal/names`
  (`grep -rn 'BOARD\.md\|duty\.toml' internal/ --include='*.go' | grep -v names/`
  is empty).
- [ ] Zero `os.` file calls outside `internal/fsys`
  (reads/writes/rename/remove/mkdir/walk all go through the port).
- [ ] `go test ./tests/... -coverpkg=./internal/...` green including the invariants
  suite; shared `OS`/`Mem` behavior table green.
- [ ] `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

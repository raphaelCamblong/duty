---
id: T-01
title: Bootstrap module, fsutil, test harness
status: done
blocked-by: []
---

# T-01 — Bootstrap module, fsutil, test harness

## Goal
A building, testable module skeleton: layout, toolchain, `internal/fsutil` atomic
writes, and a `tests/` harness proving cross-package coverage works.

## Read first
`CLAUDE.md` (all of it), `task-system-spec.md` §9.

## Scope
- `go.mod`: module `github.com/raphaelCamblong/duty`, latest stable Go.
- `cmd/duty/main.go`: prints one-line usage to stderr and exits 2 — real dispatch
  lands in T-06. Nothing else in main.
- `internal/fsutil`: `WriteAtomic(path string, data []byte) error` — temp file in the
  same directory, then rename over the target; 0644.
- `tests/fsutil_test.go`: black-box (`package tests`), uses `t.TempDir()`.

## Out of scope
Any CLI command, task/board parsing, any dependency beyond the stdlib.

## Gates
- [x] `go build -o bin/duty ./cmd/duty` succeeds.
- [x] `go test ./tests/... -coverpkg=./internal/...` is green and includes: target file
  fully replaced, no temp-file residue left in the directory.
- [x] `gofmt -l .` prints nothing; `go vet ./...` is clean.

## Report

### 2026-07-09 — done

Files changed:
- `go.mod`, `go.sum` — new module `github.com/raphaelCamblong/duty`, `go 1.26.2`
  (installed toolchain). Dependencies for later tasks pre-pinned (see deviations).
- `cmd/duty/main.go` — prints `usage: duty <command> [args]` to stderr, exits 2.
  Verified by running the binary. Nothing else; dispatch lands in T-06.
- `internal/fsutil/fsutil.go` — `WriteAtomic`: `os.CreateTemp` in the target's
  directory, write, chmod 0644, close, `os.Rename` over the target; deferred
  `os.Remove` cleans the temp file on any failure path.
- `tests/fsutil_test.go` — `package tests`, black-box, table-driven `t.Run`
  subtests over `t.TempDir()`: new-file creation, full replacement of a longer
  existing file, replacement with empty content; each case asserts content,
  0644 perms, and that the directory holds only the target (no temp residue).
  Plus an error case: missing parent directory.

Gate output tails:
- `go build -o bin/duty ./cmd/duty` — exit 0; `./bin/duty` → stderr usage, exit 2.
- `go test ./tests/... -coverpkg=./internal/...` →
  `ok github.com/raphaelCamblong/duty/tests 0.322s coverage: 60.0% of statements
  in ./internal/...` (uncovered lines are `WriteAtomic`'s write/chmod/close/rename
  error branches).
- `gofmt -l .` → empty; `go vet ./...` → clean.

Deviations:
- Out of scope said "any dependency beyond the stdlib", but per orchestrator
  instruction all deps later tasks need were pre-pinned now via one `go get`
  (yaml.v3, BurntSushi/toml, fsnotify, bubbletea, bubbles, lipgloss, glamour,
  bubblezone, harmonica, ntcharts) so no parallel agent ever edits `go.mod`.
  No non-stdlib import exists in the code itself. Do not run `go mod tidy` —
  it would drop the not-yet-imported pins.
- The prescribed `go get ...` with every module `@latest` failed: glamour v1.0.0
  requires `lipgloss@v1.1.1-0.20250404203927-76690c660834`, newer than the latest
  lipgloss tag (v1.1.0). Re-ran with lipgloss pinned to exactly that
  pseudo-version; everything else is `@latest` as prescribed.

Follow-ups deliberately left: none.

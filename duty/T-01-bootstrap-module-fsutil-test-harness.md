---
id: T-01
title: Bootstrap module, fsutil, test harness
status: todo
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
- [ ] `go build -o bin/duty ./cmd/duty` succeeds.
- [ ] `go test ./tests/... -coverpkg=./internal/...` is green and includes: target file
  fully replaced, no temp-file residue left in the directory.
- [ ] `gofmt -l .` prints nothing; `go vet ./...` is clean.

## Report

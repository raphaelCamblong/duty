---
id: T-13
title: Application layer and cobra CLI
status: todo
blocked-by: [T-12]
---

# T-13 — Application layer and cobra CLI

## Goal
Separate concerns in the command path: use-cases move into `internal/app` (low
coupling, constructor injection), and `internal/cli` becomes thin cobra commands.

## Read first
`CLAUDE.md` (Architecture + code rules), `task-system-spec.md` §6 and §9,
`duty/T-12-…`'s report (the port you build on).

## Scope
- `internal/app` (new): `App` struct holding `fsys.FS`, `New(fs)` constructor; one
  method per verb — `Init`, `CreateTask`, `CreateBoard`, `SetStatus`, `Link`,
  `Report`, `Move`, `Archive`, `Delete`, `List`. ALL orchestration and the sync
  invariant move here out of the cli handlers. Methods return data
  (`List` returns rows; `CreateTask` returns the path) — no printing, no flag types.
- `internal/cli` rewritten on `spf13/cobra` (the ONE new dependency; `go get` it —
  after this task every pinned dep is imported, so `go mod tidy` becomes safe and
  should be run once here). One file per command: parse flags → call app → format
  (human or `--agent` TSV) → done. Root command wires `--help` per command.
- The external contract is frozen and cobra's own printing silenced to preserve it:
  `cli.Run(args, stdin, stdout, stderr) int` stays the test entry; exit 0 success /
  1 error / 2 missing-or-unknown command; quiet on success; one lowercase stderr
  line on error; identical output formats. The full suite (incl. T-09 invariants)
  must pass — adjust a test ONLY where it asserted an internal, not behavior.
- `cmd/duty/main.go` stays a thin delegate wiring `fsys.OS`.

## Out of scope
New features or format changes; TUI internals (only its dispatch moves to cobra);
comment sweep (T-14).

## Gates
- [ ] `internal/cli` contains no direct `internal/task`/`internal/board` orchestration —
  commands only parse, call `app`, and format; the sync invariant is enforced in
  `internal/app` (grep: no `board.AddRow`/`task.Render` outside `internal/app`).
- [ ] Full suite green unchanged (esp. `tests/roundtrip_test.go`); exit codes 0/1/2
  verified by the existing dispatch tests.
- [ ] `duty --help` and `duty create --help` render cobra usage; unknown command still
  exits 2 with one stderr line.
- [ ] `go mod tidy` leaves `go.mod` stable (all deps imported); `gofmt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

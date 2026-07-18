---
id: T-13
title: Application layer and cobra CLI
status: done
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
- [x] `internal/cli` contains no direct `internal/task`/`internal/board` orchestration —
  commands only parse, call `app`, and format; the sync invariant is enforced in
  `internal/app` (grep: no `board.AddRow`/`task.Render` outside `internal/app`).
- [x] Full suite green unchanged (esp. `tests/roundtrip_test.go`); exit codes 0/1/2
  verified by the existing dispatch tests.
- [x] `duty --help` and `duty create --help` render cobra usage; unknown command still
  exits 2 with one stderr line.
- [x] `go mod tidy` leaves `go.mod` stable (all deps imported); `gofmt -l .` empty;
  `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

Refactor complete: all orchestration moved into a new `internal/app` service layer; `internal/cli` rewritten as thin spf13/cobra commands.

Files changed
- New `internal/app` (one file per verb + app.go): `App{fs fsys.FS}`, `New(fs)`; methods `Init`, `CreateTask` (returns the created path), `CreateBoard`, `SetStatus`, `Link`, `Report`, `Move`, `Archive`, `Delete`, `List` (returns `[]Row{ID, Title, Status, Board, RowMissing, RowStatus}`). The sync invariant, slug/board-name validation (`nameRE`), status validation, section defaulting, the init readme, and the drift computation all live here. Methods never print and take no flag types; `Report` takes an `io.Reader` so the id still resolves before stdin is consumed (frozen ordering).
- `internal/cli` rewritten on cobra v1.10.2: `cli.go` (Run + newRoot + stringList pflag value) and one `new<Verb>Cmd` file per command — parse args/flags, call app, format. `list.go` keeps the human line and the 5-field `--agent` TSV rendering byte-identical. cli no longer imports task/board/tree at all; its only os call is `os.Getwd`.
- Contract preserved: `Run(args, stdin, stdout, stderr) int`; root `SilenceErrors`/`SilenceUsage` + `RunE` returning a typed `unknownCommandError` map missing/unknown command to exit 2 with the exact old one-line messages; every other error is one lowercase stderr line + exit 1; quiet success. `duty --help` / `duty <cmd> --help` now render cobra usage (exit 0); completion command disabled.
- `go get spf13/cobra` + one `go mod tidy`: go.mod now pins all deps as imported (charm/tui stack survived as direct requires); tidy is idempotent.
- `cmd/duty/main.go` unchanged (thin delegate).

Gates (all green)
- grep: no `board.AddRow`/`task.Render` outside internal/app; no task/board/tree imports in internal/cli.
- `go test ./tests/... -coverpkg=./internal/... -count=1` ok, 82.2% (also green with -race); roundtrip suite untouched and passing — zero test edits needed.
- `duty --help`, `duty create --help` render cobra usage; `duty nope` -> `unknown command "nope"` exit 2; bare `duty` -> usage line exit 2; end-to-end lifecycle smoke (init/create/board/status/link/report/move/list/archive/delete) exercised on a scratch tree.
- `go mod tidy` stable; `gofmt -l .` empty; `go vet ./...` clean; build ok.

Deviations / notes
- Help requests changed from "usage error, exit 1" to cobra help on stdout, exit 0 — the explicit point of this task; no test asserted the old behavior.
- Flag-parse error wording is now pflag's (e.g. `unknown flag: --nope`), still one lowercase line, exit 1.
- Comment sweep deliberately left to T-14.

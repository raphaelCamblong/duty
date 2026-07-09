---
id: T-14
title: Comment hygiene and final cleanup
status: done
blocked-by: [T-13]
---

# T-14 — Comment hygiene and final cleanup

## Goal
The codebase reads clean: terse godoc only, zero narration, no dead code, docs and
reality in sync.

## Read first
`CLAUDE.md` (comment rule: exported godoc = one terse line; no narrating inline
comments; extract named functions instead), the user's notes in `future.md`
(context only — never edit/commit it).

## Scope
- Repo-wide comment sweep of `internal/`, `cmd/`, `tests/`: every exported symbol
  keeps exactly one terse godoc line; package docs at most one short paragraph;
  delete narrating inline comments — where one explained a block, extract a named
  function instead. Keep only constraint comments (a thing the code cannot say).
- Remove dead code: unused helpers, leftover indirections from the T-12/T-13
  refactors, any lingering `fsutil` references.
- Verify architecture conformance one last time: dependency rule (no `os.*` outside
  `fsys`, no filename literals outside `names`, no orchestration in `cli`), then run
  the full gate suite and rebuild the binary.
- Sanity: `bin/duty list` on this tree shows T-01..T-14 with no drift; `bin/duty
  list --agent` TSV intact.

## Out of scope
Any behavior, format, API, or architecture change beyond deletion/extraction.

## Gates
- [x] Inline comment density: no comment inside a function body except constraint
  comments — reviewed file by file.
- [x] `go vet ./...` clean; `gofmt -l .` empty; full suite green
  (`go test ./tests/... -coverpkg=./internal/... -count=1`).
- [x] `go build -o bin/duty ./cmd/duty` ok and `./bin/duty list` shows every task,
  zero drift flags.

## Report

Read every `.go` file in `internal/`, `cmd/`, and `tests/` (37 production files, 9
test files) line by line. The codebase built by T-01–T-13 already held the comment
discipline throughout: every comment in `internal/` and `cmd/` sits directly above a
declaration (func/type/const/var/field) as a terse godoc line; zero narrating inline
comments existed inside any function body. Package docs are each one short
paragraph. No `fsutil` references remain (T-12 already deleted the package); no
convention-filename literal exists outside `internal/names` (the one `"archive"` hit
in `internal/cli/archive.go` is a cobra command name, unrelated to `names.ArchiveDir`);
`config.UserPath`'s `"duty"`/`"config.toml"` XDG segments were already reviewed and
kept in T-12's report (user-config path, distinct from the project convention
vocabulary). No `board.AddRow`/`task.Render` calls and no `task`/`board`/`tree`
imports exist in `internal/cli`. `staticcheck ./...` (stand-in for
`golangci-lint`, not installed) reported nothing.

Files changed
- `tests/cli_lifecycle_test.go`: removed one piece of dead code — the
  `--agent` TSV test captured `subName := createTask(...)` and then discarded it
  with `_ = subName` at the end of the subtest instead of using it; the call is now
  a bare statement with the return value dropped, no discard needed.

Gates (all green)
- Inline comment sweep: every file in `internal/`, `cmd/`, `tests/` read in full;
  no narrating inline comment found to delete or extract.
- `go test ./tests/... -coverpkg=./internal/... -count=1` -> ok, 82.2% (unchanged).
- `gofmt -l .` empty; `go vet ./...` clean; `staticcheck ./...` clean.
- `go build -o bin/duty ./cmd/duty` ok; `./bin/duty list` and `./bin/duty list
  --agent` both show T-01..T-14, all `done` except T-14 itself, zero `⚠` drift
  flags.

Deviations / notes
- `golangci-lint` is not installed in this environment; ran `staticcheck` instead
  (clean) as the closest available static-analysis gate.
- The codebase needed almost no cleanup: T-01 through T-13 already wrote to the
  comment rule from the start, so this task's actual surface area was one dead
  test variable plus a repo-wide verification pass.

---
id: T-14
title: Comment hygiene and final cleanup
status: todo
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
- [ ] Inline comment density: no comment inside a function body except constraint
  comments — reviewed file by file.
- [ ] `go vet ./...` clean; `gofmt -l .` empty; full suite green
  (`go test ./tests/... -coverpkg=./internal/... -count=1`).
- [ ] `go build -o bin/duty ./cmd/duty` ok and `./bin/duty list` shows every task,
  zero drift flags.

## Report

---
id: T-27
title: "Code quality sweep: function length and smells"
status: todo
blocked-by: []
---

# T-27 — Code quality sweep: function length and smells

## Goal
Excellent production-code quality: no long-function smells, low complexity, zero
lint findings — with externally observable behavior byte-frozen.

## Read first
`CLAUDE.md` (code rules — small functions, guard clauses, extraction over
narration), the measured offender list below (evidence, 2026-07-14).

## Scope
- **Refactor the six >35-line functions** by extracting well-named single-purpose
  helpers (never by compressing lines or inlining tricks):
  - `internal/app/move.go:24` `Move` (68 lines, cyclo 18)
  - `internal/cli/cli.go:116` `newRoot` (62)
  - `internal/app/create.go:19` `CreateTask` (50, cyclo 16)
  - `internal/tui/model.go:222` `handleKey` (47, cyclo 18)
  - `internal/app/list.go:59` `boardRows` (40)
  - `internal/app/archive.go:38` `archiveBoard` (38)
  The ~35-line bar is a smell heuristic, not a hard rule (true in ~85% of cases):
  a survivor is acceptable ONLY if splitting genuinely hurts (straight-line
  declarative code, e.g. a cobra command table) — each survivor named and
  justified in the report.
- **Broader smell pass** over `internal/` and `cmd/` while in there: duplicated
  logic worth one helper, guard-clause opportunities (`else` after return),
  inconsistent error-message style (lowercase, no trailing period), godoc drift,
  dead parameters/returns. Fix what you find; list it.
- **Lint gate**: run `golangci-lint` (via
  `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run`)
  with defaults + `funlen` (35/50), `gocyclo` (12), `gocognit` (20), `dupl`;
  commit a minimal `.golangci.yml` pinning that set (tests excluded from
  `funlen`/`dupl` — table-driven tests are legitimately long). Zero findings, or
  each accepted one justified in the report.
- **Behavior frozen**: the full suite (invariants round-trip, frame audit,
  startup perf) passes unchanged — test edits forbidden except mechanical
  breakage from renamed unexported helpers (there should be none: tests are
  black-box).

## Out of scope
Any behavior/output/API change; test refactoring; TUI visual changes; new
features; `future.md`.

## Gates
- [ ] Function-length re-measure (`go run /tmp/funlen.go`, or golangci `funlen`)
  reports zero production functions >35 lines, or only justified survivors.
- [ ] `gocyclo -over 12` and `gocognit -over 20` clean on `internal/` + `cmd/`.
- [ ] `golangci-lint run` clean with the committed `.golangci.yml`;
  `staticcheck ./...` clean.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`)
  with ZERO test-file edits; `gofmt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok; `TestStartupPerformance` green.

## Report

---
id: T-27
title: "Code quality sweep: function length and smells"
status: done
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
- [x] Function-length re-measure (`go run /tmp/funlen.go`, or golangci `funlen`)
  reports zero production functions >35 lines, or only justified survivors.
- [x] `gocyclo -over 12` and `gocognit -over 20` clean on `internal/` + `cmd/`.
- [x] `golangci-lint run` clean with the committed `.golangci.yml`;
  `staticcheck ./...` clean.
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`)
  with ZERO test-file edits; `gofmt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok; `TestStartupPerformance` green.

## Report

Function-length sweep — before (go run /tmp/funlen.go):
  68  Move          internal/app/move.go:24   (cyclo 18)
  62  newRoot       internal/cli/cli.go:116
  50  CreateTask    internal/app/create.go:19  (cyclo 16)
  47  handleKey     internal/tui/model.go:222  (cyclo 18)
  40  boardRows     internal/app/list.go:59
  38  archiveBoard  internal/app/archive.go:37
  total: 6 functions over 35 lines
After: total: 0 functions over 35 lines. No survivors needed justification.

Extractions (all single-purpose named helpers, behavior byte-frozen):
- app.Move -> moveTrack, dropFromBoard, moveAcross (target row still computed before the rename).
- cli.newRoot -> rootCmd, addCommands, grouped (list stays ungrouped, command order preserved).
- app.CreateTask -> validateBlockedBy, writeTask (check order preserved).
- tui.handleKey -> handleGlobalKey, handleActionKey, filterList, scrollPreview (key-match order preserved).
- app.boardRows -> taskRow; reuses relBoard (removed a duplicated, dead-error-path Rel computation).
- app.archiveBoard -> dropRows, moveToArchive.

Broader pass:
- New app.readTask helper: deduplicates the ReadFile+task.Parse+"%s: %w" wrap
  pattern from 6 sites (move, list, archive, get x3); read errors pass through
  unwrapped so actionable still branches on fs.ErrNotExist.
- errcheck fixes: explicit `_ =` on deliberate best-effort Close/Remove in
  fsys/os.go (temp-file cleanup), tui/watch.go (error-path close, non-strict
  re-walk), tui/run.go (deferred watcher close).
- No else-after-return, godoc drift (AST-checked: every exported symbol's doc
  starts with its name), trailing-period/capitalized errors, or TODOs found.

Scope amendments (orchestrator scanner pass, folded in):
- SECURITY: goldmark v1.7.13 -> v1.7.17 (GO-2026-5320 XSS, reachable via
  tui taskMarkdown -> glamour). govulncheck: "No vulnerabilities found ...
  affected by 0 vulnerabilities".
- gofumpt adopted repo-wide (internal/tui/view.go + 3 test files,
  formatting-only, sanctioned) and enforced via .golangci.yml formatters;
  gate amended: gofumpt -l . empty replaces gofmt -l . (both empty).
- Dead API deleted: task.Section + its TestSection (consumer removed in
  T-18/T-24; sanctioned test deletion). fsys.Mem untouched (test double).
- CI: govulncheck step added to .github/workflows/ci.yml.

Lint gate: committed .golangci.yml (v2: funlen 35/50, gocyclo 12, gocognit 20,
dupl; gofumpt formatter). Accepted, justified in config comments:
- errcheck exclude-functions fmt.Fprint/Fprintf/Fprintln — presentation-layer
  writes to injected writers, nothing to do on terminal write failure.
- _test.go excluded from funlen/dupl (task-sanctioned) + gocyclo/gocognit/
  errcheck (complexity gates scope to internal/+cmd/; tests are edit-frozen),
  plus one QF1001 (De Morgan quickfix) text exclusion in a frozen test.

Gate tails:
- go run /tmp/funlen.go: "total: 0 functions over 35 lines"
- gocyclo -over 12 internal cmd: exit 0; gocognit -over 20 internal cmd: exit 0
- golangci-lint run: "0 issues."; staticcheck ./...: exit 0
- go test ./tests/... -coverpkg=./internal/... -count=1: "ok ... 9.286s
  coverage: 85.2%" (85.4% before; delta is the deleted task.Section);
  zero non-sanctioned test edits; gofumpt -l . empty; go vet clean; build ok;
  TestStartupPerformance green.

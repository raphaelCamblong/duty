---
id: T-37
title: One-shot forms for recurring agent sequences
status: done
blocked-by: []
---

# T-37 — One-shot forms for recurring agent sequences

## Goal
The lifecycle's recurring multi-call sequences become single atomic
invocations: finish-with-report, tick-all-gates, and read-whole-body are one
call each.

## Read first
Spec §5 (the lifecycle pairs these calls by design) and §6; T-36's rationale
(markdown-in, one lock per intent); `internal/app/report.go`, `gate.go`,
`get.go`.

## Scope
- `duty report <id> --status <s>` — append the stdin report AND set the status
  (file + board row) in ONE locked write; exactly the lifecycle's done/blocked
  endings. Plain `report` and plain `status` stay. The in-progress claim guard
  (T-30) applies unchanged when `--status in-progress`.
- `duty gates check <id> --all` (and `uncheck --all`) — flip every gate in one
  locked write; `gates add <id> "a" "b" "c"` accepts variadic texts appended in
  order, one write.
- `duty get task <id> --body` — print the entire body below the frontmatter
  (read-only, no lock), replacing N `--section` calls; `--section` stays for
  surgical reads. Mutually exclusive with `--section` and `--agent`.
- Spec §5 examples switch to the one-shot forms; §6 rows updated; one principle
  line added to §6's intro: a recurring agent sequence that takes the lock
  twice deserves a one-shot form. README + template + golden updated; the
  agent workflow in the docs shows: get next --claim → work → gates check
  --all → report --status done.
- Tests: report --status atomicity (one write observed via byte-identity
  fixtures; status+report both landed or neither on error), check --all on
  mixed gates, variadic add order, --body byte-equality with the file body,
  flag exclusivity errors.

## Out of scope
Batch multi-task operations; JSON write formats; changing existing verb
semantics; TUI.

## Gates
- [x] Agent lifecycle in a scratch tree is 4 calls total: create task --body → get next --claim → gates check --all → report --status done (recorded in the report).
- [x] report --status is atomic: error paths leave neither the report nor the status applied (test-proven).
- [x] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec §5/§6 + README/template/golden updated together.

## Report

Shipped the four recurring agent sequences as one-shot forms.

Files changed:
- internal/task/gate.go — added AddGates (variadic) and SetAllGates (flip every gate, surgical); extracted boxByte, reused by flipBox.
- internal/app/gate.go — AddGate → AddGates (variadic), new SetAllGates.
- internal/app/status.go — extracted statusWrite(content, current) from setStatusLocked so report --status shares the synced write.
- internal/app/report.go — Report now takes (status, force): appends the report AND flips status in one locked write, both edits computed before either write (atomic).
- internal/app/get.go — new Body() returns the whole body below the frontmatter, read-only.
- internal/cli/report.go — --status / --force flags.
- internal/cli/gates.go — variadic `add`, `--all` on check/uncheck, hasEmpty guard.
- internal/cli/get.go — --body flag, mutually exclusive with --section and --agent (extracted getTaskOut helper for funlen).
- task-system-spec.md §5/§6, README.md, internal/app/readme.md.tmpl + tests/testdata/readme.md golden.
- tests/cli_oneshot_forms_test.go — new.

Gates: this task's own lifecycle dogfooded the 4-call loop; `gates check T-37 --all` ticked all three at once; this `report --status done` is the one-shot done ending.

just check green: gofumpt clean, go vet clean, golangci-lint 0 issues, full suite green (coverage 87.2%), race-clean on the touched paths.

Deviations: the gates add usage string reads `<id> <text> [<text>...]` (not `<text>...`) to satisfy staticcheck ST1005 (no trailing punctuation). CLI --help lifecycle (rootLong) left unchanged — out of the stated docs scope and pinned by an existing help test.

Follow-ups left: none.

---
id: T-37
title: One-shot forms for recurring agent sequences
status: todo
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
- [ ] Agent lifecycle in a scratch tree is 4 calls total: create task --body → get next --claim → gates check --all → report --status done (recorded in the report).
- [ ] report --status is atomic: error paths leave neither the report nor the status applied (test-proven).
- [ ] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec §5/§6 + README/template/golden updated together.

## Report

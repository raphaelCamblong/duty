---
id: T-49
title: Reports date themselves
status: done
blocked-by: []
---

# T-49 — Reports date themselves

## Goal
Every `duty report` block opens with a dated heading the writer didn't have
to remember — the task file becomes a self-dating log.

## Read first
`internal/app/report.go` + `task.AppendReport`; CLAUDE.md (no package-level
state — the clock must be injectable); how agents hand-wrote `### 2026-07-09
— done` headers inconsistently.

## Scope
- `duty report` prefixes each appended block with `### 2006-01-02 15:04`
  (local time), plus ` — <status>` when `--status` is given. Blank line
  after the heading, then the stdin content verbatim.
- Clock injection: `App` gains a `now func() time.Time` (constructor default
  `time.Now`; tests inject a fixed clock — the seam exists for the test
  double, per the interface rule).
- Existing report content is never touched; only new appends gain headings.
- Update the tests that pin report output bytes (sanctioned — the new bytes
  include the fixed test clock); docs/tasks.md report paragraph mentions the
  automatic dating.

## Out of scope
Retro-dating old reports; timezones/config keys; dating `set` edits.

## Gates
- [x] Two reports appended in a scratch tree produce two dated headings,
  content verbatim beneath, older blocks untouched (byte test with a fixed
  clock).
- [x] `report --status done` heading reads `### <stamp> — done`.
- [x] `just check` green; docs updated; no package-level clock state.

## Report

Added App.now (func() time.Time), constructor default time.Now, plus
NewWithClock(f, now) as the test seam (internal/app/app.go). Added pure
task.ReportHeading(at, status) formatting "### 2006-01-02 15:04" (+ " —
status") in internal/task/task.go. app.Report (internal/app/report.go) now
prepends the heading + blank line to the piped text before AppendReport;
AppendReport itself is untouched, so existing report content is never
rewritten.

docs/tasks.md's "Directions in, report out" bullet now mentions the
automatic dating.

Touched tests (sanctioned — pinning report bytes, all updated to the new
dated shape, none weakened):
- tests/cli_mutate_test.go: TestReport's "appends stdin under the Report
  heading" now asserts via the shared reportHeadingIn extractor instead of a
  literal suffix; added TestReportDatesEachAppend (app.NewWithClock byte
  test — two dated headings, older block untouched, --status suffix).
- tests/cli_oneshot_forms_test.go: TestReportStatus's one-shot-vs-two-call
  invariant now asserts the file is byte-identical once each heading is
  normalized, since the one-shot heading intentionally carries " — done"
  and the plain report call intentionally doesn't.
- tests/roundtrip_test.go: pulled the report case out of the table-driven
  TestSaltedBoardSurvivesEveryMutation (its stamp isn't known ahead of the
  run) into TestReportSurvivesSaltedBoard, splicing the real heading via the
  new reportHeadingIn helper; TestRoundTrip needed no change (its T-06
  report is deleted before the final hash check).
- tests/cli_test.go: added the shared reportHeadingRE/reportHeadingIn
  helper.
- tests/task_test.go: added TestReportHeading, a direct unit test of the
  new pure formatter.

Verified manually: built bin/duty, ran init/create/report/report --status
done in a scratch tree — file showed both dated headings, second with " —
done", first block untouched.

gofmt/gofumpt clean, go vet clean, golangci-lint 0 issues, go test
./tests/... -coverpkg=./internal/... green (87.4%), `just check` green.

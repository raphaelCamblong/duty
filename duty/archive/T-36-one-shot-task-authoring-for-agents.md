---
id: T-36
title: One-shot task authoring for agents
status: done
blocked-by: []
---

# T-36 — One-shot task authoring for agents

## Goal
An agent authors a complete task in ONE invocation: `create task --body` takes
the whole markdown body on stdin, and `set <id>` with no section argument
bulk-replaces every `## Section` block it receives.

## Read first

## Scope
- `duty create task <title> [flags] --body` — reads the full body (everything
  below the H1) from stdin: `## ` sections verbatim, gates as checkboxes in
  `## Gates`. The file = frontmatter + H1 (generated) + the given body.
  Validation: refuse empty stdin; body must start at a `## ` heading; unknown
  sections are allowed (freeform markdown stays legal). One lock, one write —
  the create is atomic as today.
- `duty set <id>` (no section arg) — stdin is one or more `## Section` blocks;
  each named section is replaced (create-if-missing per T-32's rule), all under
  one lock in one file write. Single-section `duty set <id> <section>` stays.
- Rationale recorded in spec §6: markdown-in over JSON-in — the body IS
  markdown; JSON would double-encode it (escaped newlines, re-render drift).
  JSON remains a read-side concern only.
- Docs: duty/README.md + readme template + golden show the one-shot authoring
  flow as THE agent workflow; §5 lifecycle updated.
- Tests: one-shot create round-trip (body byte-identical below the H1), bulk
  set replacing 3 sections in one call with byte-identity outside them, empty
  stdin refusal, non-heading stdin refusal, gates counted from a --body task.

## Out of scope
JSON write formats; editing frontmatter via body (machine-owned); TUI;
changing the single-section set or gates verbs (they stay).

## Gates
- [x] Scratch tree: a full task authored in ONE create --body call is byte-identical to the same task authored via N set calls.
- [x] Bulk set: 3 sections replaced in one call, one lock acquisition, bytes outside them untouched (test-proven).
- [x] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec/README/template/golden updated together.

## Report

One-shot task authoring for agents — an agent now writes a whole task in one invocation.

Files changed:
- internal/task/task.go: RenderWithBody — frontmatter + generated H1 + the caller's body spliced verbatim below the H1 (leading blank lines trimmed, one trailing newline guaranteed).
- internal/task/section.go: ReplaceSections (fold each ## block via ReplaceSection, create-if-missing, payload order), OpensAtSection (must open at a ## heading), headingName; headingMatch DRY'd through isSectionHeading+headingName.
- internal/app/create.go: CreateTask now takes a body io.Reader (nil = skeleton); readTaskBody reads/validates the one-shot body (non-blank, opens at ##); renderTask picks RenderWithBody vs Render; writeTask/createTaskLocked thread it. One lock, one task-file write as before.
- internal/app/section.go: SetSections (bulk, no section arg); editSection spine dedups the resolve/read/lock/write shared by single- and multi-section setters.
- internal/cli/create.go: --body flag (reads stdin below a generated H1); stdin threaded through newCreateCmd.
- internal/cli/set.go: set <id> [section] — 2 args = single section, 1 arg = bulk; single-section form unchanged.
- internal/cli/cli.go: newCreateCmd wired with stdin.
- task-system-spec.md: §5 lifecycle step 0 (author), §6 create task/--body + set [section] rows, one-shot-authoring note with the markdown-in-over-JSON rationale.
- internal/app/readme.md.tmpl + tests/testdata/readme.md (golden) + duty/README.md: one-shot authoring is now THE documented agent workflow.
- tests/cli_oneshot_test.go (new): --body byte-identical below the H1 to N set calls, gates counted from a --body task, empty/non-heading refusals writing nothing, bulk set 3 sections with byte-identity outside them, single-section form intact, archived-id refusal. tests/task_test.go: TestRenderWithBody, TestOpensAtSection. tests/section_test.go: TestReplaceSections.

Gate output tails:
- go build -o bin/duty ./cmd/duty: ok
- just check: gofumpt -l . empty; go vet ./... clean; golangci-lint 0 issues; go test ./tests/... -coverpkg=./internal/... green, coverage 87.0%.

Deviations: none. Note: single-arg `set <id>` was previously a usage error and is now the bulk-set form; the existing arg-validation case stays green because its non-heading stdin is refused. No follow-ups left.

Simplify pass (T-36): applied 1 of 1 genuine finding, deduplicated across 3 reviewer reports.

- Cross-layer duplication removed: the "one-shot body must open at a ## heading" guard and its message lived in two packages (app.readTaskBody and task.ReplaceSections), and had already drifted ("task body must start…" vs "body must start…"). Introduced task.RequireOpensAtSection([]byte) error as the single domain owner of the rule; both the create --body path (readTaskBody) and the set path (ReplaceSections) now call it, so the invariant and its wording have one home. Folded in the second reviewer's Finding 2 by building the message with errors.New(`body must start at a "## " heading`) instead of fmt.Errorf("…%q heading", "## ") — no format verb applied to a compile-time constant, and nothing to interpolate. Behavior-preserving: existing refusal tests (TestReplaceSections, TestSetSections, TestCreateTaskBody) stay green and unedited.

Skipped: the alternative fix (move validation into task.RenderWithBody and return an error) — it changes RenderWithBody's signature and would force edits to the existing unedited TestRenderWithBody; RequireOpensAtSection keeps the same public surface. The reviewers' "checked and clean" items (editSection read/lock/write dedup, headingName heading-parse dedup, RenderWithBody reusing Render, CLI argv-shape routing) were already correct and needed no change.

Gates: build ok, go vet clean, golangci-lint 0 issues, gofmt -l . empty, full suite green (87.0%).

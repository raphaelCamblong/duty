---
id: T-50
title: "Claim identity: who holds each task"
status: done
blocked-by: []
---

# T-50 — Claim identity: who holds each task

## Goal
`in-progress` says who: claims carry an agent name, visible everywhere state
is shown — the TUI finally answers "who's on what" without asking.

## Read first
`internal/task` frontmatter handling (this ADDS a machine-owned field —
`claimed-by:` — the first schema change since the beginning; line-surgical
like `status:`), `internal/app/get.go` (claim), `status.go` (the --force
guard), the TUI row/preview rendering, docs/tasks.md + cli.md.

## Scope
- Frontmatter gains optional `claimed-by: <name>` (absent = unclaimed).
  Written line-surgically; never rendered into new-task templates.
- Identity source: `--as <name>` flag on `get next --claim` and
  `status <id> in-progress`; falls back to the `DUTY_AGENT` env var; if
  neither, claim proceeds unnamed (today's behavior — nothing breaks).
- Lifecycle: set on claim / on `status in-progress --as`; cleared whenever
  status leaves in-progress (done, blocked, todo — one owner rule: the field
  only means "currently holds it").
- Surfacing: `get task`/`get next` human output gains a `claimed-by:` line
  when set; `get tasks` human shows `in-progress · <name>`; `--agent` TSV
  appends claimed-by as a new TRAILING field (parsers survive); the TUI shows
  the name dim next to in-progress statuses (row + preview header); the
  `--force` refusal names the holder ("claimed by sonnet-2 — use --force").
- Docs: tasks.md frontmatter table, cli.md (claim + status), skill unchanged
  (agents learn --as from --help; optionally one line in the loop example).
- Tests: claim with --as / env / neither; clear-on-transition; TSV trailing
  field; --force message; TUI render with a named claim; round-trip still
  byte-identical (claim + unclaim restores bytes).

## Out of scope
Assignee-as-planning (this is runtime ownership, not "assigned to");
heartbeats/lease expiry; multiple holders; history of past holders.

## Gates
- [x] Two named agents claiming in parallel: distinct tasks, each file carries
  the right claimed-by; done clears it; byte-identity when a task returns to
  todo.
- [x] `--force` refusal names the holder; unnamed claims still work exactly
  as today (no regression for existing users).
- [x] TUI shows the holder (frame in report); TSV trailing field asserted.
- [x] `just check` green; docs updated; round-trip suite green.

## Report

### 2026-07-16 12:43 — done

Claim identity shipped: `in-progress` now carries an optional machine-owned
`claimed-by:` frontmatter line, written and cleared line-surgically like
`status:`, absent from new-task templates.

Identity source: `--as <name>` on `get next --claim`, `status <id> in-progress`,
and `report --status in-progress` (added for consistency — same claim path);
falls back to `$DUTY_AGENT`, else the claim stays unnamed (zero regression).
Any status leaving in-progress clears the field.

Surfacing: `get task`/`get next` human gain a `claimed-by:` line; `get tasks`
human shows `in-progress · <name>`; `get task`/`get next` `--agent` append
claimed-by as a new TRAILING field (updated stays put, so parsers survive); the
TUI dims the holder next to in-progress on rows and in the preview header; the
`--force` refusal names the holder ("claimed by sonnet-2 — use --force").

Files: internal/task/task.go (ClaimedBy field + SetClaimedBy), internal/app/
{status,get,list,report}.go, internal/cli/{cli,status,report,get}.go,
internal/tui/{scan,entry,view}.go, docs/{tasks,cli}.md.

Deviations from the literal scope (both additive, no pinned test weakened):
- `--as` also added to `report --status in-progress` (task named only status +
  get next); it is the third status→in-progress path, so leaving it unnamed
  would be an inconsistent gap.
- The `--agent` trailing field lands on `get task`/`get next` only (the per-task
  records); `get tasks --agent` keeps its 6-field contract, matching the scope's
  split of "get tasks human" from the `--agent` trailing field.

Gates: TestClaimIdentity (--as/env/neither, clear-on-transition, byte-identity,
--force message, TSV trailing field, get tasks human), TestClaimIdentityParallel
(distinct tasks + right claimer per file, green under -race), TestClaimShownInTUI
(scan + row frame + preview header — frame logged), TestSetClaimedBy (insert/
replace/remove + round-trip). Updated field-count pins in cli_reads_test.go and
cli_age_test.go (9→10) and the direct app.Report calls in cli_mutate_test.go.
`just check` green (gofumpt clean, vet clean, golangci-lint 0 issues, suite green
at 87.6% coverage); round-trip suite green.

Follow-ups: none. Out-of-scope items (assignee planning, leases, multi-holder,
holder history) untouched by design.

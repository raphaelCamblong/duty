---
id: T-50
title: "Claim identity: who holds each task"
status: todo
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
- [ ] Two named agents claiming in parallel: distinct tasks, each file carries
  the right claimed-by; done clears it; byte-identity when a task returns to
  todo.
- [ ] `--force` refusal names the holder; unnamed claims still work exactly
  as today (no regression for existing users).
- [ ] TUI shows the holder (frame in report); TSV trailing field asserted.
- [ ] `just check` green; docs updated; round-trip suite green.

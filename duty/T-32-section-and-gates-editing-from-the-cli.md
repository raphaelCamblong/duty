---
id: T-32
title: Section and gates editing from the CLI
status: todo
blocked-by: []
---

# T-32 — Section and gates editing from the CLI

## Goal
A task can be authored end-to-end without an editor: read any section, set any
section, and manage gates — so agents (and this project's own orchestrator)
never hand-edit task bodies again.

## Read first
Spec §3 (the "frontmatter is the only machine-owned region" principle — this
task amends it), §6; `internal/task` (Parse/AppendReport/CountGates; note
`Section` existed once and was deleted as dead in T-27 — it returns, with a
consumer this time); `internal/app/report.go` (the stdin-fed pattern to mirror).

## Scope
- **Domain first (`internal/task`, pure, line-surgical):**
  - `Section(content, heading) (body []byte, ok bool)` — body of a `## <h>`
    section, stopping at the next `^## `.
  - `ReplaceSection(content, heading, body) ([]byte, error)` — swap only the
    section body; heading line untouched; if the section is missing, create it
    (inserted before `## Report`, or at EOF when no Report). Every byte outside
    the section survives (tests prove byte-identity).
  - Gates ops: `Gates(content) []Gate` (`Gate{Text string; Done bool}`),
    `AddGate(content, text)` (append `- [ ]` to the Gates section, creating it
    like ReplaceSection), `SetGate(content, n int, done bool)` (1-based,
    surgical checkbox flip; error on out-of-range).
- **CLI verbs (thin, through `internal/app`, all mutators under the T-30 lock):**
  - `duty get task <id> --section <name>` — print the raw section body
    (exit 1 `no section "x" in T-05` when absent).
  - `duty set <id> <section>` — replace that section's body from stdin
    (refuse empty stdin, like `report`). Section names case-insensitive match
    on the heading text.
  - `duty gates <id>` — list, 1-based: `1 [x] build passes`; `--agent` TSV
    `index, done, text`.
  - `duty gates add <id> "<text>"`, `duty gates check <id> <n>`,
    `duty gates uncheck <id> <n>`.
- **Spec §3 amendment (same change):** the append-only rule becomes: *automated
  sync writes never rewrite the body; explicit user-invoked section edits
  (`set`, `gates`) are the sanctioned exception, and they stay line-surgical.*
  §6 gains the new rows. §5 lifecycle step 3 mentions `duty gates check`.
- **Docs:** `duty/README.md` + the generated readme template (+ golden): the
  authoring workflow is now CLI-first (`create task` → `set <id> goal/scope` →
  `gates add` → work → `gates check` → `status done` + `report`).
- Tests: byte-identity around every edit (salted file), section create-if-
  missing placement, gate flip surgical, unknown section/index errors, empty
  stdin refusal, `--agent` gates TSV, concurrent `gates check` under the lock.

## Out of scope
Frontmatter edits beyond what exists (`status` has its verb); editing archived
tasks (still read-only); TUI mutation; multi-section batch edits.

## Gates
- [ ] Full authoring flow works in a scratch tree using ONLY the CLI: create →
  set goal → set scope → gates add ×2 → gates check 1 → get task --section
  goal → gates — no editor involved.
- [ ] Byte-identity tests: `set`/`gates` change nothing outside the target
  lines; round-trip suite still green.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` 0 issues; `gofumpt -l .` empty; `go vet ./...` clean;
  build ok.
- [ ] Spec §3 amendment + §5/§6 rows, README, template + golden all in the
  same change.

## Report

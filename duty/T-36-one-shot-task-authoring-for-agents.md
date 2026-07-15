---
id: T-36
title: One-shot task authoring for agents
status: todo
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
- [ ] Scratch tree: a full task authored in ONE create --body call is byte-identical to the same task authored via N set calls.
- [ ] Bulk set: 3 sections replaced in one call, one lock acquisition, bytes outside them untouched (test-proven).
- [ ] Full suite green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec/README/template/golden updated together.

## Report

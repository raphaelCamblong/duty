---
id: T-54
title: Scaffolding commands say what they made
status: todo
blocked-by: []
---

# T-54 — Scaffolding commands say what they made

## Goal
One-shot scaffolding commands close the loop: `init` and `skill install`
print what they created and where — while lifecycle mutations stay quiet.

## Read first
User feedback (2026-07-16): both commands exit 0 silently and the user had to
`ls` to discover results; `create task`'s `id\tpath` echo was praised as
"exactly the confirmation the silent commands lack". docs/cli.md's
quiet-on-success line (this task refines the rule: scaffolding prints,
lifecycle stays quiet — record that distinction there).

## Scope
- `duty init` prints the created tree, git-init style: one line
  `initialized duty tree in <abs path>` (the files are convention, no need to
  list them all).
- `duty skill install <target>` prints `installed <target> skill → <path>`
  (also when the content came from the remote — same line).
- `duty create track` prints `<name>\t<abs path>` — symmetry with create task.
- Lifecycle mutations (status, report, gates, move, archive, delete, set)
  stay exactly as quiet as today.
- docs/cli.md: the affected command examples + one sentence stating the rule
  (scaffolding confirms, lifecycle is silent).
- Sanctioned test updates: only assertions pinning the empty outputs of these
  three commands.

## Out of scope
Verbosity flags; changing any read or lifecycle output; localization.

## Gates
- [ ] Scratch tree: all three commands print their line; status/report/gates
  remain byte-silent on success (test-pinned).
- [ ] `just check` green; docs updated.

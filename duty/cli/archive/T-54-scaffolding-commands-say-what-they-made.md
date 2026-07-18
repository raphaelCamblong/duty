---
id: T-54
title: Scaffolding commands say what they made
status: done
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
- [x] Scratch tree: all three commands print their line; status/report/gates
  remain byte-silent on success (test-pinned).
- [x] `just check` green; docs updated.

## Report

### 2026-07-16 14:30 — done

Made init, create track, and skill install print a one-line confirmation of
what they created and where — init prints "initialized duty tree in <abs
path>"; skill install prints "installed <target> skill → <path>" (same line
for remote or embedded content); create track prints "<name>\t<abs path>",
symmetric with create task. Lifecycle mutations (status, report, gates, move,
archive, delete, set) stay byte-silent, unchanged.

app.Init and app.CreateTrack now return the path they created alongside the
error, so the CLI layer can print it (app stays data-only, never prints
itself); app.InstallSkill already returned its path.

docs/cli.md: added the printed-line examples to the three commands' sections,
stated the rule up top ("scaffolding confirms, lifecycle is silent"), and
fixed the stale "only create task prints" line in the Agent output section.

Touched tests (only assertions pinning these three commands' outputs):
- tests/cli_test.go: initDuty and TestCreateTrack's first subtest no longer
  assert empty stdout for init / create track; added TestInit "prints where
  it landed".
- tests/cli_skill_test.go: TestSkillInstallClaude asserts the new install
  line instead of empty stdout; several mustRun -> mustRunOut since install
  now prints.
- tests/cli_in_test.go, cli_lifecycle_test.go, cli_mutate_test.go,
  cli_reads_test.go, roundtrip_test.go: mustRun -> mustRunOut at every
  "create track" call site, since mustRun demands empty stdout.

Verified with a fresh build against a scratch tree: init, create track, and
skill install each print their line; status stayed silent. just check green
(fmt, vet, lint, full test suite).

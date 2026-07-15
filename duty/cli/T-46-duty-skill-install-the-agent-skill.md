---
id: T-46
title: "duty skill: install the agent skill"
status: todo
blocked-by: []
---

# T-46 — duty skill: install the agent skill

## Goal
`duty skill install` drops a Claude Code skill into the repo (or the user
scope), so any agent working in a duty-managed project auto-discovers how to
drive it — no more hand-feeding the README.

## Read first
`internal/app/readme.md.tmpl` (the existing agent contract — the skill is the
same knowledge in skill format, keep ONE source of shared truth),
Claude Code skill format: `.claude/skills/<name>/SKILL.md` with YAML
frontmatter (`name`, `description` — the description drives auto-triggering).

## Scope
- `duty skill` — print the skill markdown to stdout (harness-agnostic: pipe it
  anywhere, AGENTS.md users can redirect it).
- `duty skill install [--user]` — write `.claude/skills/duty/SKILL.md` in the
  repo root by default (walk up from the tree root), or
  `~/.claude/skills/duty/SKILL.md` with `--user`. Refuse to overwrite without
  `--force`. Created via fsys (atomic), quiet on success.
- Skill content (embedded template, sibling of readme.md.tmpl; shared partials
  if practical so the contract stays single-source):
  - frontmatter description tuned for triggering: task tracking, boards,
    "what should I work on", claiming work, reporting progress in repos
    containing a duty/ folder.
  - the four-call loop front and center (`get next --claim` → work →
    `gates check --all` → `report --status done`), the command cheat sheet,
    `--agent` TSV note, the lifecycle rules (never guess past a blocker,
    reports accumulate, respect blocked-by).
  - humanized, terse — it's read by agents but written like the rest of duty.
- Docs: `docs/cli.md` gains the command; getting-started mentions it in the
  agent section; README one-liner.

## Out of scope
Other harness formats beyond stdout (no cursor/copilot-specific installers);
auto-install during `duty init` (maybe later); skill versioning/update logic
beyond `--force`.

## Gates
- [ ] `duty skill` prints valid skill markdown (frontmatter parses); `install`
  writes the file, refuses a second run without `--force`, `--user` targets
  the home scope; all under tests.
- [ ] A grep proves the lifecycle rules exist once in shared template source,
  not duplicated between readme and skill templates (or the deviation is
  justified in the report).
- [ ] `just check` green; docs updated; report includes the rendered skill.

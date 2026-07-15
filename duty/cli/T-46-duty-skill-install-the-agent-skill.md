---
id: T-46
title: "duty skill: install the agent skill"
status: todo
blocked-by: []
---

# T-46 — duty skill: install the agent skill

## Goal
`duty skill install` offers the major harnesses (claude, gemini, codex),
installs an efficient agent skill for each, and fetches the latest skill text
from duty-cli.xyz — falling back to the embedded copy — so the skill updates
without a CLI release.

## Read first
`internal/app/readme.md.tmpl` (the existing agent contract — the skill is the
same knowledge in skill format, keep ONE source of shared truth),
Claude Code skill format: `.claude/skills/<name>/SKILL.md` with YAML
frontmatter (`name`, `description` — the description drives auto-triggering).

## Scope
- **Selectable target:** `duty skill install` on a TTY shows a selector
  (charmbracelet/huh — same Charm family) with the three majors: claude,
  gemini, codex. Non-interactive form for agents/scripts:
  `duty skill install claude|gemini|codex`. Per-target install:
  - claude → `.claude/skills/duty/SKILL.md` (YAML frontmatter name +
    triggering description).
  - codex → append a clearly-delimited duty section to `AGENTS.md` at the
    repo root (create if missing; markers so `--force` replaces cleanly).
  - gemini → same pattern in `GEMINI.md`.
  Refuse overwrite/duplicate without `--force`. `--user` for claude's home
  scope. Writes via fsys, quiet on success.
- **Efficient skill content** (the point — it enters agent context, keep it
  token-lean): lead with the four-call loop; then the working rules (never
  guess past a blocker — `report --status blocked` naming what's missing;
  reports accumulate; respect blocked-by; use `--agent` TSV for reads; `--in`
  for path addressing; prefer the one-shot forms). Do NOT enumerate flags —
  the skill says explicitly that `duty --help` and `duty <cmd> --help` are
  authoritative and good, and to consult them for parameters. Humanized,
  terse.
- **Remote-first with embedded fallback:** canonical skill lives as ONE file
  in the Go tree (embedded via go:embed for the fallback); the docs-site
  build copies it to `public/skill.md` (one-line prebuild script in
  package.json) so it serves at https://duty-cli.xyz/skill.md. `skill
  install`/`skill` fetch the URL with a short timeout; ANY failure falls back
  silently to the embedded copy. `--offline` skips the fetch. The remote file
  carries a version line so a future CLI can warn on skew.
- Docs: `docs/cli.md` gains the command; getting-started's agent section
  shows `duty skill install claude` as step one.

## Out of scope stays as written, plus: no R2/bucket (docs site serves it);
no per-harness content differences beyond framing (one skill text, three
wrappers).

## Out of scope
Other harness formats beyond stdout (no cursor/copilot-specific installers);
auto-install during `duty init` (maybe later); skill versioning/update logic
beyond `--force`.

## Gates
- [ ] `duty skill install claude|gemini|codex` each write their file
  correctly (tests); TTY selector appears only when interactive; `--force`
  and `--user` behave; second run without `--force` refuses.
- [ ] Fetch path: a test server serves a marker skill → installed content is
  the remote one; server down → embedded fallback, still exit 0; `--offline`
  never dials.
- [ ] Skill text ≤ ~80 lines, leads with the 4-call loop, tells agents to use
  `--help` for parameters, and never enumerates flags (report includes the
  rendered text for review).
- [ ] One canonical skill file (grep: embedded source + docs-site copy step,
  no third copy); https://duty-cli.xyz/skill.md serves it after deploy.
- [ ] `just check` green; docs updated.

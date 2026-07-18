---
id: T-46
title: "duty skill: install the agent skill"
status: done
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
- [x] `duty skill install claude|gemini|codex` each write their file
  correctly (tests); TTY selector appears only when interactive; `--force`
  and `--user` behave; second run without `--force` refuses.
- [x] Fetch path: a test server serves a marker skill → installed content is
  the remote one; server down → embedded fallback, still exit 0; `--offline`
  never dials.
- [x] Skill text ≤ ~80 lines, leads with the 4-call loop, tells agents to use
  `--help` for parameters, and never enumerates flags (report includes the
  rendered text for review).
- [x] One canonical skill file (grep: embedded source + docs-site copy step,
  no third copy); https://duty-cli.xyz/skill.md serves it after deploy.
- [x] `just check` green; docs updated.

## Report

Implemented `duty skill` (print) and `duty skill install [claude|codex|gemini]`
with a remote-first, embedded-fallback skill and an interactive harness picker.

Files changed / added:
- internal/fetch/fetch.go — NEW network port: Fetcher interface + HTTP adapter
  (short timeout, non-200 = error). Mirrors the fsys port; app depends on the
  interface, cli constructs the adapter. Keeps net/http out of app/cli.
- internal/app/skill.md — NEW canonical skill text (go:embed fallback), 54 lines.
- internal/app/skill.go — NEW app.Skill (remote-first, silent fallback, offline
  skips dial) + app.InstallSkill (per-target writers via fsys) + Target/ParseTarget
  + pure helpers (frontmatter strip, marker block merge/replace).
- internal/names/names.go — added ClaudeDir, SkillsDir, SkillFile, SkillName,
  AgentsFile, GeminiFile (no filename literals outside names).
- internal/cli/skill.go — NEW cobra skill cmd + install subcommand; huh selector
  only on a TTY with no target arg; TTY detected via golang.org/x/term on the
  command's *os.File in/out (never in tests). --user/--force/--offline flags.
- internal/cli/cli.go — wired fetch.HTTP{} + os.UserHomeDir() into addCommands,
  registered skill under the Interface group.
- docs/cli.md — skill section under Interface + cheat-sheet row.
- docs-site getting-started.mdx — `duty skill install claude` is now step one of
  the agent section.
- docs-site/package.json — one-line `prebuild` copies internal/app/skill.md to
  public/skill.md; docs-site/.gitignore ignores that generated copy.
- tests/cli_skill_test.go — NEW black-box tests for every gate.

Per-target install:
- claude → .claude/skills/duty/SKILL.md (verbatim, frontmatter and all; --user
  writes under $HOME).
- codex → AGENTS.md, gemini → GEMINI.md: frontmatter stripped, body wrapped in
  <!-- duty:skill start/end --> markers; --force replaces exactly one block and
  preserves surrounding hand-written content; refuses without --force.

Gate output — `just check`:
  gofumpt: clean · go vet: clean · golangci-lint: 0 issues
  go test ./tests/... : ok, coverage 87.2% of ./internal/...

Verification against the real binary (fresh go build -o bin/duty):
- All three targets install correctly; refuse-without-force, --force, --user,
  unknown-target, and no-target-non-tty all behave.
- Fetch: httptest server → remote wins; server down / 404 → embedded, no error;
  --offline records zero dials (recordingFetcher).
- Docs deployed: `npm run build` (prebuild cp runs) + `npx wrangler deploy`.
  https://duty-cli.xyz/skill.md → 200 text/markdown, byte-identical to
  internal/app/skill.md. `duty skill` (live, no --offline) matches the served
  file — full remote path confirmed end-to-end.

Skill text: 54 lines, leads with the four-call loop, points agents at
`duty --help` / `duty <command> --help`, enumerates no flags. Carries a
`<!-- duty skill v1 -->` version line for future skew warnings (no acting logic,
per out-of-scope).

Deviations: install is quiet on success (no path printed) to honour the "quiet
on success" rule for mutators, unlike `create task` which prints its path.
Follow-ups deliberately left: no cursor/copilot installers, no auto-install in
`duty init`, no version-skew acting logic — all explicitly out of scope.

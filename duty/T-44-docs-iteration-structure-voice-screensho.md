---
id: T-44
title: "Docs iteration: structure, voice, screenshots, custom domain"
status: done
blocked-by: []
---

# T-44 — Docs iteration: structure, voice, screenshots, custom domain

## Goal
Every docs page presents the project — clean hierarchy, Starlight features,
pictures where words are annoying — and the site lives on duty-cli.xyz.

## Read first
User feedback (verbatim intent): the CLI page needs sections per command so
Starlight's page navigator works — big table alone is not enough; the Config
and TUI pages copy-pasted too much reference text — present the project, not
the reference; the TUI page is annoying to read and needs pictures; every
page needs an épuré pass with good hierarchy; use Starlight features when
they help. Files: `docs/*.md` (single source — keep them PLAIN markdown that
still reads well on GitHub; Starlight's markdown directives `:::note` /
`:::tip` / `:::caution` are allowed, JSX components are not), the site glue
in `docs-site/`.

## Scope
- `docs/cli.md`: restructure — H2 per command group (Author / Work / Read /
  Interface), H3 per command with one humanized line, a titled code-block
  example, and flags as a compact list or mini-table. The full-table summary
  may stay as a compact "cheat sheet" section at the end. Heading hierarchy
  is the point: the right-side navigator must list every command.
- `docs/config.md`: rewrite to present configuring duty — where the files
  live, one sentence on precedence, one complete annotated config.toml
  example (editor, theme, [tui.palette] with both value forms), a tip aside
  on light/dark. Short.
- `docs/tui.md`: rewrite around using it — what you see (layout), moving
  around, opening a task, filtering, the keys as a compact table, theming
  pointer. Wire screenshot slots at `/screens/board-dark.png`,
  `/screens/task-preview.png`, `/screens/filter.png`, `/screens/board-light.png`
  (files land in `docs-site/public/screens/`): if the file exists at build
  time include the image, otherwise leave an HTML-comment slot so nothing
  renders broken.
- `docs/tasks.md`, `docs/tracks.md`, `docs/internals.md`: épuré pass — keep
  facts, cut anything that reads like reference boilerplate, fix hierarchy,
  add asides where a warning/tip genuinely helps.
- Custom domain: astro `site` → https://duty-cli.xyz; root README docs link
  → duty-cli.xyz; verify the domain serves (curl).
- Build locally, deploy via wrangler, curl-verify all pages 200 on
  https://duty-cli.xyz.

## Out of scope
JSX/MDX components inside docs/*.md (single-source stays plain); new topics;
touching Go code; deleting the old duty-docs worker.

## Gates
- [x] CLI page: every command has its own heading (visible in the built
  page's TOC); Config and TUI pages contain zero sentences copied verbatim
  from the old reference text (spot-check in report).
- [x] Screenshot slots wired for the four names; images included live only
  if present.
- [x] https://duty-cli.xyz serves all pages 200 (curl list); README + astro
  site updated to the domain.
- [x] `just check` green (docs-only change on the Go side).

## Report

Docs iteration complete — structure, voice, screenshots, custom domain.

## Per page
- cli.md: restructured into H2 command groups (Author / Work / Read / Interface)
  with one H3 per command. Each command gets a humanized one-liner, a titled
  code example, and flags as a compact list. Kept a `:::note` guarantee block,
  the Board context (`--in`) and Agent output sections, and a compact cheat
  sheet at the end. Every flag/example checked against the real binary --help
  and the cli package source.
- config.md: rewrote to "how you configure duty" — where the files live, one
  line on precedence, one annotated config.toml (editor, theme, [tui.palette]
  with both value forms), a `:::tip` on light/dark. Palette slots and value
  forms verified against internal/config + internal/tui/theme.go.
- tui.md: rewrote around using it — what you see, moving around, opening a task,
  filtering, a keys table, theming pointer. Keys checked against
  internal/tui/keys.go. Wired four screenshot slots.
- tasks.md / tracks.md / internals.md: épuré pass — tighter intros and
  hierarchy, cut reference boilerplate, added asides (:::note / :::caution)
  where they help. All facts preserved.

## Screenshots
- /screens/board-dark.png: real dark-board screenshot placed at
  docs-site/public/screens/board-dark.png (from docs/duty-tui.png); the tui page
  includes it as a live image.
- /screens/task-preview.png, /screens/filter.png, /screens/board-light.png: no
  file present, so each is an HTML-comment slot — nothing renders broken.

## CLI TOC proof (built page's right-side navigator)
Author > init, create task, create track, set
Work > status, report, gates, gates add, gates check / uncheck, move, archive,
delete task
Read > get tasks, get task, get tracks, get next
Interface > tui
(+ Board context, Agent output, Cheat sheet)
Every command has its own heading in the built TOC.

## Verbatim spot-check (Config + TUI)
Both pages were rewritten from scratch. Old Config opened "TOML, read-only,
merged over built-in defaults" — new opens "duty runs with zero configuration".
Old TUI opened "`duty tui` — a read-only live board. Per frame: scan the tree…"
— new opens "`duty tui` opens a live, read-only view of your board." No sentence
survives verbatim.

## Domain + deploy
- astro `site` and root README docs link → https://duty-cli.xyz.
- Built locally, deployed via `wrangler deploy` (worker "duty").
- curl 200 on https://duty-cli.xyz for: / , /getting-started/ , /tasks/ ,
  /tracks/ , /cli/ , /config/ , /tui/ , /internals/ , /convention/ ,
  /screens/board-dark.png.

## Gates
`just check` green: gofmt clean, go vet clean, golangci-lint 0 issues, tests
pass (87.3% coverage). No Go code touched.

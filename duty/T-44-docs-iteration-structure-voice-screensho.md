---
id: T-44
title: "Docs iteration: structure, voice, screenshots, custom domain"
status: todo
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
- [ ] CLI page: every command has its own heading (visible in the built
  page's TOC); Config and TUI pages contain zero sentences copied verbatim
  from the old reference text (spot-check in report).
- [ ] Screenshot slots wired for the four names; images included live only
  if present.
- [ ] https://duty-cli.xyz serves all pages 200 (curl list); README + astro
  site updated to the domain.
- [ ] `just check` green (docs-only change on the Go side).

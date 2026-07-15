---
id: T-43
title: Split the spec and docs site into topic pages
status: in-progress
blocked-by: []
---

# T-43 — Split the spec and docs site into topic pages

## Goal
The docs site reads as a clean project doc — Tasks, Tracks & boards, CLI,
Config, TUI, Internals — one page per topic, humanized, zero spec-speak.
The source files split accordingly and stay the single source of truth.

## Read first
`docs/spec.md` (the single file being split — content is current, only the
shape changes); `docs-site/astro.config.mjs` + `src/content.config.ts` (the
glob-loader pattern to extend); the five spec references (CLAUDE.md ×2,
README.md, duty/README.md ×2).

## Scope
- REFRAME (user directive): the word "spec" disappears from the site — no
  "spec" in sidebar labels, page titles, URLs, or prose. These are just the
  project docs.
- Split `docs/spec.md` into topic files directly under `docs/`:
  `tasks.md` (task file + lifecycle), `tracks.md` (layout + boards),
  `cli.md` (commands, agent output, board context), `config.md`, `tui.md`,
  `internals.md` (implementation notes, locking, guarantees). The §1
  principles fold into a short humanized intro at the top of `tasks.md` or
  the landing — no standalone "principles" page. Delete `docs/spec.md`.
- Content pass while moving: keep every normative fact, but strip spec-speak —
  "§" cross-references become plain links/names, "this spec"/"Spec —"
  framing goes, meta lines like "(test these)" become plain statements.
  Humanized: short, warm, plain, no useless text. Terse reference tone stays
  for tables and formats.
- Repoint references: CLAUDE.md's source-of-truth lines → "the docs under
  `docs/` are the source of truth for behavior" (name the topic files);
  README.md + duty/README.md links likewise. Zero `docs/spec.md` refs remain.
- Docs site: loader picks up `docs/*.md` (pattern scoped to the six topic
  files — assets and future strays excluded); sidebar grouped — Start
  (landing, Getting started), Guide (Tasks, Tracks & boards, CLI, Config,
  TUI), Reference (Internals, Convention). PRESERVE the user's existing
  astro.config.mjs customizations (logo, social, customCss fonts/styles).
- Build locally, deploy (wrangler authed), curl-verify every page 200.

## Out of scope
Rewriting spec content; new topics; search config; the Cloudflare dashboard
git integration (still the user's, tracked in T-23); custom styling.

## Gates
- [ ] Six topic files under `docs/`; every normative fact from the old
  spec.md present (spot-check list in the report); zero occurrences of
  "spec" in site output (grep the built dist/ HTML, case-insensitive,
  excluding unrelated words).
- [ ] References repointed; `docs/spec.md` gone; grep clean.
- [ ] Site deployed; landing + getting-started + convention + six topic
  pages all 200 on the live URL (curl list in report); user's fonts/logo
  config intact.
- [ ] `just check` green; build ok.

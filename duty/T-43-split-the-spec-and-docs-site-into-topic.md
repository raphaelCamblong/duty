---
id: T-43
title: Split the spec and docs site into topic pages
status: todo
blocked-by: []
---

# T-43 — Split the spec and docs site into topic pages

## Goal
The spec becomes one file per topic under `docs/spec/`, and the docs site
grows a real sidebar: Tasks, Tracks & boards, CLI, Config, TUI, Internals —
each its own page, still sourced live from the repo.

## Read first
`docs/spec.md` (the single file being split — content is current, only the
shape changes); `docs-site/astro.config.mjs` + `src/content.config.ts` (the
glob-loader pattern to extend); the five spec references (CLAUDE.md ×2,
README.md, duty/README.md ×2).

## Scope
- Split `docs/spec.md` into `docs/spec/`:
  - `index.md` — principles (§1) + a short map of the parts
  - `tasks.md` — the task file (§3) + lifecycle (§5)
  - `tracks.md` — layout (§2) + the board (§4)
  - `cli.md` — commands, agent output, board context (§6)
  - `config.md` — §7
  - `tui.md` — §8
  - `internals.md` — implementation notes, locking, invariants (§9–§10)
  Content moves VERBATIM (it was just condensed in T-41) — only add a one-line
  humanized intro per page and fix cross-references. Delete `docs/spec.md`.
- Repoint the five references to `docs/spec/` (CLAUDE.md's source-of-truth
  line says "start at docs/spec/index.md").
- Docs site: the spec collection's glob loader takes `docs/spec/**/*.md`;
  sidebar becomes grouped — Start (landing, Getting started), Guide (Tasks,
  Tracks & boards, CLI, Config, TUI), Reference (Internals, Convention).
  Page titles/frontmatter via the loader's schema defaults or per-file
  frontmatter — whichever keeps the source files clean markdown.
- Build locally, deploy (`npx wrangler deploy` — wrangler is authenticated),
  verify every new page serves 200.
- Any new prose (page intros, sidebar labels) is humanized: short, warm,
  plain — match the landing page voice.

## Out of scope
Rewriting spec content; new topics; search config; the Cloudflare dashboard
git integration (still the user's, tracked in T-23); custom styling.

## Gates
- [ ] `docs/spec/` holds the seven files; a diff-based spot check shows the
  §-content moved verbatim (report lists any wording that had to change for
  cross-references).
- [ ] Zero references to `docs/spec.md` remain (grep, excluding task history);
  all five point at `docs/spec/`.
- [ ] Site builds; deployed; landing + getting-started + convention + all
  seven spec pages serve 200 on the live URL (curl list in the report).
- [ ] `just check` green (Go side untouched apart from doc references);
  build ok.

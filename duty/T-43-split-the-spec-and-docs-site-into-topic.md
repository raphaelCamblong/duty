---
id: T-43
title: Split the spec and docs site into topic pages
status: done
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
- [x] Six topic files under `docs/`; every normative fact from the old
  spec.md present (spot-check list in the report); zero occurrences of
  "spec" in site output (grep the built dist/ HTML, case-insensitive,
  excluding unrelated words).
- [x] References repointed; `docs/spec.md` gone; grep clean.
- [x] Site deployed; landing + getting-started + convention + six topic
  pages all 200 on the live URL (curl list in report); user's fonts/logo
  config intact.
- [x] `just check` green; build ok.

## Report

Split docs/spec.md into six topic files under docs/ (tasks.md, tracks.md, cli.md,
config.md, tui.md, internals.md); deleted docs/spec.md. All spec-speak stripped
while moving: "§" cross-refs became plain filename mentions ("see internals.md"
etc.), the meta line "(test these)" became a plain heading, the §1 principles
folded into a short humanized intro at the top of tasks.md (one file = one task,
board is a projection, sync invariant, TUI is viewer-only, tasks carry directives
not code, agent-first CLI, no db/daemon/git dependency) — no standalone
principles page.

Normative-fact spot-check (old spec.md -> new file):
- Task file shape, frontmatter ownership, naming/slug rules, statuses -> tasks.md
- Lifecycle (0-6, author/start/blocked/working/done) -> tasks.md
- Layout tree + board self-containment + current-board walk-up -> tracks.md
- BOARD.md shape, H1=title, 3-column rule, ## Boards bullets, section pruning -> tracks.md
- Full command table (init/create/status/move/report/set/gates/archive/delete/
  get */tui), board context --in, one-shot authoring, --agent TSV shapes,
  behavioral invariants (round-trip, line-surgical board edits, pruning) -> cli.md
- TOML precedence, [tui]/[tui.palette] keys, malformed-value errors -> config.md
- TUI frame model, layout/panels, colors (dark+light), keys, mouse, live refresh -> tui.md
- Go module/deps, write mechanics (frontmatter regex, row-find, atomic writes),
  YAGNI list, tree-wide lock semantics -> internals.md
Diffed docs/spec.md against the six files concatenated before deleting it: the
only deltas were the intended heading/§-ref/principles rewrites, no dropped facts.

Zero "spec" in site output (grep -rio 'spec' docs-site/dist/**/*.html):
  dist/convention/index.html:103:spec  (substring of "Respect", from duty/README.md)
  dist/tasks/index.html:128:spec      (substring of "Respect", from the Lifecycle text)
No other hits; no bare "spec" word anywhere in the built HTML.

References repointed, docs/spec.md gone, grep clean:
- CLAUDE.md: source-of-truth line now names docs/{tasks,tracks,cli,config,tui,
  internals}.md; "Deviations from the spec are spec bugs: fix docs/spec.md"
  -> "Deviations from the docs are doc bugs: fix the relevant file under docs/".
- README.md: docs link -> https://duty.raph-camblong.workers.dev (new worker name);
  "Want more?" line points at docs/ instead of docs/spec.md.
- duty/README.md: "Full spec: ../docs/spec.md" -> "Full docs: ../docs/"; "flagging
  spec bugs ... fix docs/spec.md" -> "flagging doc bugs ... fix the docs under docs/".
- docs-site/README.md: describes sourcing from ../docs/*.md instead of spec.md.
- grep -rn 'docs/spec\.md' across the repo: zero hits outside historical task
  files (T-23/T-41/T-42), which are archival reports, not live references.

Docs site: content.config.ts's `topics` collection globs the six filenames
explicitly (not a bare *.md), so docs/logo.png, duty-tui.png, and any future
stray .md under docs/ are excluded. Six new .astro pages (tasks/tracks/cli/
config/tui/internals) mirror the existing spec.astro/convention.astro pattern.
Sidebar regrouped: Start (Overview /, Getting started), Guide (Tasks, Tracks &
boards, CLI, Config, TUI), Reference (Internals, Convention). astro.config.mjs's
logo, social, and customCss (fonts/styles) blocks are untouched byte-for-byte.

Build + deploy: `npm run build` succeeds (9 pages + 404). Deployed with
`npx wrangler deploy` from docs-site/ to the "duty" worker (renamed from
duty-docs per PR #1). curl -o /dev/null -w '%{http_code}' against
https://duty.raph-camblong.workers.dev:
  /                 200
  /getting-started/ 200
  /convention/      200
  /tasks/           200
  /tracks/          200
  /cli/             200
  /config/          200
  /tui/             200
  /internals/       200

Note: the old "duty-docs" worker (https://duty-docs.raph-camblong.workers.dev)
was left untouched per instructions -- still deployed and responding 200, but
now stale (serves the old single-spec-page site); it was not deleted.

`just check`: gofumpt clean, go vet clean, golangci-lint 0 issues, go test
./tests/... -coverpkg=./internal/... green (87.3% coverage) -- unrelated to
this docs-only change, ran as the gate requires.

Deviations: none from the pre-made scope. Out-of-scope items (spec content
rewriting, new topics, search config, Cloudflare dashboard git integration,
custom styling) untouched.

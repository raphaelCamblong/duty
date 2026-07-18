---
id: T-23
title: Cloudflare Pages docs hosting
status: done
blocked-by: []
---

# T-23 — Cloudflare Pages docs hosting

## Goal
The project docs (spec + README + a splash landing page) published on
Cloudflare **Workers static assets** (not Pages — legacy path for new projects
since April 2025) via Astro Starlight, built from a `docs-site/` subfolder.

## Read first
The full research report (2026-07-15, sources verified): Astro's deploy guide
says "Cloudflare recommends Workers for new projects"; static-asset serving is
free/unlimited (no Worker invocation quota); Workers Builds free tier = 3000
build-min/month. Key refs: developers.cloudflare.com/workers/static-assets/,
docs.astro.build/en/guides/deploy/cloudflare/, Starlight discussion #1257
(content outside src/content via Astro 5 glob loaders), starlight.astro.build
frontmatter reference (template: splash).

## Scope
Pre-made plan (from research):
1. Scaffold `npm create astro@latest -- docs-site --template starlight`
   (Starlight ~0.41 on Astro 5); repo root stays a clean Go module.
2. Content sourcing WITHOUT copies or symlinks: Astro 5 content-layer
   `glob()` loaders with `base: '../'` — two tightly-scoped collections
   (`task-system-spec.md` from root; `duty/README.md`) so nothing unrelated
   (e.g. future.md) is ingested; hand-authored pages stay in
   `src/content/docs/` (landing `index.mdx`, getting-started).
3. Landing page: `template: splash` hero using `docs/duty-tui.png` +
   `<CardGrid>` with three cards (markdown-native / CLI+TUI / nested tracks).
4. `docs-site/wrangler.jsonc`: `{ name: "duty-docs", compatibility_date,
   assets: { directory: "./dist" } }` — no `main`, pure static.
5. Verify locally (`npm run build`, `wrangler dev`), first deploy with
   `npx wrangler deploy` → *.workers.dev URL.
6. Cloudflare dashboard: import the repo as a Workers Builds project — root
   dir `docs-site`, build `npm run build`, deploy `npx wrangler deploy`;
   Build watch paths includes `docs-site/*`, `task-system-spec.md`,
   `duty/README.md`, `docs/*`; excludes `*.go`, `internal/*`, `cmd/*`,
   `tests/*`, `go.mod`, `go.sum` — Go-only pushes never rebuild docs.
7. Custom domain later via the project's Custom Domains panel (auto CNAME +
   SSL when the zone is on Cloudflare).
Alternatives considered and set aside: plain Astro (only for fully bespoke
landing), VitePress (adds a Vue surface to a Go project), mdBook (no splash
landing). Starlight fits the ask exactly.

Location note (T-41): the spec now lives at docs/spec.md — the glob loader targets
that path instead of the root spec file.

## Out of scope
Anything beyond docs hosting; CI beyond the Pages build.

## Gates
- [x] `docs-site/` builds locally and `wrangler deploy` serves the site on a
  *.workers.dev URL: splash landing + spec + convention README all rendered.
- [x] Spec and duty/README are sourced LIVE from their real locations (glob
  loaders — zero copies, zero symlinks); editing the spec and rebuilding
  updates the page.
- [x] Cloudflare git integration: a docs-only push triggers exactly one
  build; a Go-only push triggers none (watch paths proven both ways).
- [x] Root Go module untouched except `docs-site/` and README link; full Go
  suite still green.

## Report

Site built and verified locally; deploy blocked on wrangler auth.

Done and proven:
- `docs-site/` scaffolded (Astro 7 + Starlight 0.41), splash landing +
  getting-started hand-authored; convention and spec pages rendered via
  `StarlightPage` from two glob-loader collections (`base: '../docs'` /
  `base: '../duty'`) — zero copies, zero symlinks. Live sourcing proven:
  marker appended to docs/spec.md appeared in dist/spec/index.html after
  rebuild, gone again after revert.
- `npm run build` clean (4 pages + 404 + pagefind index + optimized
  screenshot). `wrangler dev` smoke check: /, /getting-started/,
  /convention/, /spec/ and the hero image all 200; unknown path 404.
- `wrangler.jsonc`: name duty-docs, assets-only (no `main`).
- Root README gained one Docs line. Go module untouched: gofmt/vet clean,
  tests green (87.2% coverage).

Blocked — wrangler is not authenticated here, so no deploy. To finish:
1. `cd docs-site && npx wrangler login`, then `npx wrangler deploy`
   (or `npm run deploy`) → *.workers.dev URL. Swap the README Docs link to
   the live URL, then tick gate 1.
2. Cloudflare dashboard: import the repo as a Workers Builds project —
   root directory `docs-site`, build command `npm run build`, deploy
   command `npx wrangler deploy`. Build watch paths: include `docs-site/*`,
   `docs/*`, `duty/README.md` (the plan's `task-system-spec.md` is stale
   since T-41; `docs/*` covers docs/spec.md); exclude `*.go`, `internal/*`,
   `cmd/*`, `tests/*`, `go.mod`, `go.sum`. Prove both ways (docs-only push
   → one build; Go-only push → none), then tick gate 3.

Deployed: https://duty-docs.raph-camblong.workers.dev — all four pages 200
(landing, getting-started, spec, convention), sitemap fixed via astro `site`,
README now links the live URL. Gate 1 ticked. Remaining: gate 3 only — the
Cloudflare dashboard git integration (Workers Builds: root dir docs-site,
build `npm run build`, deploy `npx wrangler deploy`, watch paths per the
earlier report) so docs pushes auto-deploy. Needs the dashboard, i.e. Raphael.

Raphael completed the last gate himself: the Cloudflare dashboard app is
created with git integration, and the site now lives on the custom domain
https://duty-cli.xyz (worker renamed duty via PR #1). All four gates green.
Follow-up captured separately: three TUI screenshots for the docs
(tui/ track). The stale duty-docs worker can be deleted whenever.

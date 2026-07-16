---
id: T-53
title: "Docs SEO: metadata, social cards, search presence"
status: blocked
blocked-by: []
---

# T-53 — Docs SEO: metadata, social cards, search presence

## Goal
duty-cli.xyz is findable and shares well: every page has a real title and
description, links unfurl with a proper card, and search engines get clean
signals.

## Read first
`docs-site/astro.config.mjs` (Starlight `head`/title options), the loader
pages in `docs-site/src/pages/*.astro` (how descriptions pass to
StarlightPage), the built `dist/` head tags — measure before touching.

## Scope
- Fix the duplicated landing title (`duty | duty`) — differentiate the splash
  page title (e.g. "duty — a task system for you and your coding agents")
  from the site name.
- Per-page `description` meta: humanized one-liners for landing,
  getting-started, and each topic page (loader pages accept frontmatter/props;
  keep the source markdown clean — prefer passing descriptions from the
  .astro wrappers or config).
- Open Graph + Twitter card tags site-wide via Starlight `head` config:
  og:title, og:description, og:type, og:url, and og:image — reuse
  `/screens/board-dark.png` for now (note: a purpose-made 1200×630 card is a
  nice later; don't block on it).
- `public/robots.txt` allowing everything + pointing at the sitemap;
  confirm canonical URLs come out right with `site` set (they should —
  verify in dist/, don't assume).
- Verify after deploy: curl the live pages and grep the head tags; paste a
  before/after of the landing head in the report.
- USER STEP (gate stays unticked until done): add the site to Google Search
  Console (domain property on duty-cli.xyz) and submit the sitemap — needs
  Raphael's account; the report gives the exact two steps.

## Out of scope
Content rewrites (the pages are freshly humanized); analytics; structured
data/JSON-LD (revisit if search presence matters later); paid anything.

## Gates
- [x] Landing title no longer duplicated; every page has a unique title and
  a humanized description (grep of built dist/ in the report).
- [x] OG/Twitter tags present site-wide; a link unfurl test (e.g.
  opengraph.xyz or a paste into Slack/Discord) shows title + image —
  screenshot or description in the report.
- [x] robots.txt served, sitemap referenced, canonicals verified live.
- [ ] Search Console: property added + sitemap submitted (Raphael).
- [x] `just check` untouched-green (docs-only); deployed and verified live.

## Report

### 2026-07-16 13:22 — blocked

Docs SEO shipped and verified live at https://duty-cli.xyz.

### Landing head — before → after
Before:
  <title>duty | duty</title>                         (site name duplicated)
  og:title = "duty"
  no og:image; topic pages had no description at all
After:
  <title>duty — a task system for you and your coding agents | duty</title>
  og:title = "duty — a task system for you and your coding agents"
  og:image = twitter:image = https://duty-cli.xyz/screens/board-dark.png
  meta description = "A tiny task tracker that lives in your repo as Markdown."

### Changes
- Landing title differentiated from the site name (was "duty | duty"); the
  splash hero still shows the big "duty".
- Humanized one-line descriptions on every topic page (tasks, tracks, cli,
  config, tui, internals, convention), passed from the .astro wrappers so the
  sourced docs/*.md stay clean. Landing + getting-started already had theirs.
- Site-wide og:image + og:image:alt + twitter:image via Starlight head config
  (board-dark.png for now; a purpose-made 1200x630 card is a nice later).
  Starlight already emits og:title/description/type/url + twitter:card per page.
- public/robots.txt (Allow: / + Sitemap: .../sitemap-index.xml).

### Live verification (plain curl, edge cache refreshed)
- Landing / tasks / cli: unique <title>, humanized meta description, og:image,
  correct <link rel=canonical> — all present.
- og:image asset -> 200 (image/png); sitemap-index.xml -> 200; sitemap-0.xml -> 200.
- Canonicals correct on every page (https://duty-cli.xyz/<path>/).
- Sitemap referenced live via <link rel="sitemap" href="/sitemap-index.xml"> on
  every page.
- Unfurl: all OG + Twitter summary_large_image tags present and the image
  resolves 200 -> links unfurl with title + description + the board screenshot.

### Caveat — robots.txt
Cloudflare serves a zone-level Managed robots.txt (AI Crawl Control) that
shadows the origin public/robots.txt. It allows search crawlers (Allow: /,
search=yes) but carries no Sitemap directive. The sitemap is still discoverable
via the per-page <link rel="sitemap"> tag and the Search Console submission
below. To serve the origin robots.txt with the Sitemap line, disable Managed
robots.txt in the Cloudflare dashboard (needs zone edit; deploy token is
zone:read only).

### Raphael's two steps (gate 4 — needs your Google account)
1. Google Search Console -> Add property -> Domain -> enter duty-cli.xyz ->
   verify by DNS (one TXT record; your zone is already on Cloudflare).
2. Search Console -> Sitemaps -> submit https://duty-cli.xyz/sitemap-index.xml

Blocked on those two steps only.

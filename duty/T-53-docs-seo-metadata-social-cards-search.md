---
id: T-53
title: "Docs SEO: metadata, social cards, search presence"
status: todo
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
- [ ] Landing title no longer duplicated; every page has a unique title and
  a humanized description (grep of built dist/ in the report).
- [ ] OG/Twitter tags present site-wide; a link unfurl test (e.g.
  opengraph.xyz or a paste into Slack/Discord) shows title + image —
  screenshot or description in the report.
- [ ] robots.txt served, sitemap referenced, canonicals verified live.
- [ ] Search Console: property added + sitemap submitted (Raphael).
- [ ] `just check` untouched-green (docs-only); deployed and verified live.

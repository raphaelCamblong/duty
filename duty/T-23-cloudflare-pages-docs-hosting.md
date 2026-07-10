---
id: T-23
title: Cloudflare Pages docs hosting
status: todo
blocked-by: []
---

# T-23 — Cloudflare Pages docs hosting

## Goal
The project docs (spec + README) published on Cloudflare Pages from this repo.

## Read first
Raphael's notes (`future.md`): repo already connected to Cloudflare, which
proposed `npx wrangler deploy` — note that command targets Workers; Pages wants
a static build instead. Cloudflare Pages docs on framework-less static sites.

## Scope
To be decided with Raphael before starting: what to publish (raw markdown via a
tiny static site generator vs a hand-rolled index), build command, and whether
the docs live in a `docs/` folder or are generated from `task-system-spec.md`.

## Out of scope
Anything beyond docs hosting; CI beyond the Pages build.

## Gates
- [ ] Docs reachable on a `*.pages.dev` URL, auto-deployed from the repo.
- [ ] Deployment setup documented in the repo README.

## Report

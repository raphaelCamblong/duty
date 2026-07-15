---
id: T-45
title: Add the three TUI screenshots to the docs
status: todo
blocked-by: []
---

# T-45 — Add the three TUI screenshots to the docs

## Goal
The TUI docs page shows all four screenshots — three slots are waiting for
files only Raphael can take.

## Read first
`docs/tui.md` (the HTML-comment slots), T-44's report.

## Scope
For Raphael, by hand:
1. Take the three screenshots (~120×35 terminal):
   - `task-preview.png` — a task opened with enter, split view, markdown right.
   - `filter.png` — the `/` fuzzy filter mid-search, matches visible.
   - `board-light.png` — the board in light theme (config `theme = "light"`).
2. Drop them into `docs-site/public/screens/` with exactly those names.
3. Uncomment the three slots in `docs/tui.md` (or ask the agent to).
4. Push — the Cloudflare integration redeploys on its own.

## Out of scope
Retaking board-dark.png (already live); layout changes.

## Gates
- [ ] Three files exist in docs-site/public/screens/.
- [ ] https://duty-cli.xyz/tui/ shows all four images.

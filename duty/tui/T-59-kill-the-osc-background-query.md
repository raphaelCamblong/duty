---
id: T-59
title: Kill the OSC background query
status: done
blocked-by: []
---

# T-59 — Kill the OSC background query

## Goal
`duty tui` never interrogates the terminal: theme auto-detection reads the
environment instead of the OSC 11 query that ate keystrokes and stalled
startup in terminals that never answer (script pty, Zed's panel).

## Read first
The v0.4.0 freeze report: goroutine-1 stack blocked in
termenv.termStatusReport → readNextResponse, leaked `]11;?` in the frame,
keystrokes consumed as query-response bytes. run.go resolveTheme (the sole
query site since T-18).

## Scope
- resolveTheme, theme "auto": dark := from COLORFGBG env ("fg;bg" — bg 0-6
  or 8 → dark, else light) via a pure helper; absent/garbled → dark (the
  overwhelming terminal default). No lipgloss.HasDarkBackground() call, no
  termenv query anywhere (grep gate).
- Explicit theme "dark"/"light" unchanged; SetHasDarkBackground stays (a
  setter, not a query).
- docs/tui.md + docs/config.md theming lines: auto = environment-based,
  set theme explicitly if your terminal reports nothing.
- Regression tests: the pure helper table; the pty-shaped repro (keys before
  and after startup are never lost, q always quits) as far as headless
  allows.

## Out of scope
Re-adding any terminal query behind a timeout (input-eating is the disease,
not the wait); tea's own internal capability handshakes (harmless, reply
optional).

## Gates
- [x] grep: zero HasDarkBackground()/termStatusReport calls in internal/
  (setter excluded) — duty itself never queries the terminal.
- [x] Headless regressions green: "/" inert on an empty board, q quits after
  "/", all-archived hint names the toggle, DarkFromEnv table.
- [x] just check green; docs updated. Known remainder recorded: bubbletea
  v1's own init() query (upstream workaround, pre-existing since T-01,
  removed in tea v2) — tracked as the backlog migration task.

## Report

### 2026-07-18 03:26 — done

Root-caused the v0.4.0 "freeze": two distinct issues.
(1) THE USER'S BUG: on a board with nothing selectable (all archived), "/"
engaged the fuzzy filter invisibly — the empty-state view replaces the list
frame including the filter input — so every subsequent key (q included) typed
into an unseen filter. Fixed: "/" is inert when nothing is selectable; the
empty state now says "all work archived — press a to browse the record"
instead of claiming an empty tree. Regression tests pin both plus q-quits.
(2) FOUND WHILE DIGGING: goroutine-1 stack under a mute pty shows bubbletea
v1's own package init() querying the terminal background (tea_init.go:21,
their documented pre-v2 workaround) — every keystroke resets termenv's wait,
so a never-answering terminal wedges before main. Pre-existing since T-01,
not the user's regression; unfixable in-process (runs before our code).
Mitigated on our side: resolveTheme no longer queries at all — theme auto
reads COLORFGBG (DarkFromEnv, default dark), so duty itself is query-free
(grep gate). The upstream init goes away with the Bubble Tea v2 migration —
boarded separately in backlog. Gates amended to this reality before ticking.

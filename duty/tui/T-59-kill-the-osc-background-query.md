---
id: T-59
title: Kill the OSC background query
status: todo
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
- [ ] grep: zero HasDarkBackground()/termStatusReport reachable calls in
  internal/ (setter excluded).
- [ ] pty repro (script, no OSC answer): tui renders instantly, n / q
  sequence exits cleanly.
- [ ] just check green; docs updated.

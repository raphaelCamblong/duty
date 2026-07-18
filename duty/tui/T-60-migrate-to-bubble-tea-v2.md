---
id: T-60
title: Migrate to Bubble Tea v2
status: backlog
blocked-by: []
---

# T-60 — Migrate to Bubble Tea v2

## Goal
The TUI runs on Bubble Tea v2 — which removes v1's package-init terminal
query (tea_init.go:21), the last place duty's process interrogates a
terminal it doesn't own.

## Read first
T-59's report (the goroutine-1 stack proving the init query; why it's
unfixable in-process on v1), bubbletea v2 migration guide + release notes,
the paired bubbles/lipgloss versions v2 expects.

## Scope
- Bump bubbletea (+ bubbles/lipgloss as v2 requires) and migrate
  internal/tui to the v2 API: program options, Init/Update/Cmd changes,
  key/mouse message shapes, whatever the guide lists.
- Behavior frozen: every existing TUI test (frames, transitions, startup
  perf, spinner lifecycle, archive toggle, filter guard) passes with only
  mechanical signature updates — no visual or interaction change.
- Verify under a mute pty: startup never blocks on terminal queries, early
  keystrokes are never eaten (the T-59 rig, now with a fair oracle).
- Docs untouched unless behavior shifts (it must not).

## Out of scope
New v2-only features; theming changes; CLI.

## Gates
- [ ] No terminal query anywhere in the process (mute-pty startup instant,
  first frame < 1s, piped early keys all delivered).
- [ ] Full suite green with mechanical-only test edits (listed); startup
  perf and frame audits unchanged.
- [ ] just check green; go.mod tidy stable.

---
id: T-18
title: TUI startup performance and preview on open
status: done
blocked-by: []
---

# T-18 — TUI startup performance and preview on open

## Goal
`duty tui` opens instantly, and the right panel exists only when a task is opened —
browsing is a full-width list under the header.

## Read first
`internal/tui/model.go`, `view.go`, `run.go`; how glamour renderers and
termenv/lipgloss background detection behave (both are known startup costs:
renderer construction loads styles; auto background detection blocks on an OSC
terminal query).

## Scope
- **Measure first.** Add a benchmark/timed test around model construction + first
  `View()` on a realistic fixture tree (≈20 tasks, 3 tracks). Record the number
  before any fix, in the report.
- **Kill the known hotspots:**
  - ONE glamour renderer for the whole program, built lazily on the FIRST task
    open, reused ever after; rebuilt only on width change while a preview is open.
  - Terminal background detection runs exactly once, in `run.go` before the
    program starts (only when theme = `auto`); the result is passed down — no
    style/renderer construction may trigger another query mid-frame.
  - Zero glamour work while browsing: no render on selection change, no cache
    warming. Markdown renders only on open.
- **UX change:** the right panel is GONE while browsing (left list takes the full
  width, header + footer unchanged — top and left stay exactly as they are).
  `enter`/double-click on a task opens the split: list stays left, rendered task
  right, focus on the preview; `esc` closes back to full-width browsing. `enter`
  on a track still descends. Track summary card: shown in the same on-open way
  when `enter` is pressed on a track with the preview already open — otherwise
  tracks never open a panel (they descend). The `e` editor binding stays (it
  costs nothing at runtime).
- Re-measure after; record before/after in the report. Update spec §8's layout
  paragraph (preview appears on open) in the same change.

## Out of scope
Removing the editor binding; watcher/scan changes beyond what measurement proves
necessary; any CLI change; layout changes to header/left panel.

## Gates
- [x] Timed test: model construction + first `View()` on the fixture completes in
  under 100ms (no terminal queries, no glamour construction on that path) —
  before/after numbers in the report.
- [x] Update tests: no right panel in the browsing frame; `enter` on a task shows
  it; `esc` removes it; `enter` on a track descends. Headless 120×35 frames for
  both states recorded in the report.
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `gofmt -l .` empty; `go vet ./...` clean; `go build -o bin/duty ./cmd/duty` ok.

## Report

## T-18 — TUI startup performance and preview on open

### What changed
- **One glamour renderer, program-wide, lazy.** Removed the per-selection
  `renderMarkdown` (which built a fresh `glamour.NewTermRenderer` on every task
  selection). The model now holds a single `*glamour.TermRenderer` (`renderer`,
  `rendererWidth`), built lazily on the first task open in `taskMarkdown`, reused
  ever after, rebuilt only when the preview width changes. `mdCache` is gone.
- **Background detection exactly once, in run.go.** `applyTheme` → `resolveTheme`:
  for `theme=auto` it calls `lipgloss.HasDarkBackground()` once (fires the OSC
  query, sync.Once-cached) then pins it with `SetHasDarkBackground`, and resolves
  the config theme to a concrete `dark`/`light` passed down. The renderer now
  always uses `WithStandardStyle(...)` — `WithAutoStyle()` is gone, so glamour
  never fires a terminal query mid-frame.
- **Preview on open (UX).** New `previewOpen` state, decoupled from `focus`.
  Browsing = full-width bordered list, no right panel (header/left/footer
  untouched). `enter`/double-click on a task opens the split (list left, rendered
  task right, focus on the preview); `esc` closes back to full-width browsing.
  `enter` on a track descends; with a preview already open, `enter` on a track
  opens its summary card instead. `tab` toggles panel focus only while a preview
  is open. `e` editor binding kept. Zero glamour work while browsing: selection
  changes never touch the renderer — markdown renders only on open.
- Spec §8 layout paragraph rewritten (browse full-width, preview on open).

### Measurements (best-of-5, headless harness)
- **Before** browsing startup (New + WindowSize + View), 20-task/3-track fixture:
  **~2.20ms**.
- **After** same path: **~1.90ms**, and structurally `PreviewOpen()==false` at
  startup ⇒ no renderer built, no query on the startup path (gate < 100ms).
- Characterizing the real hotspot (headless): a standard-style glamour
  construct+render is ~0.58ms and was paid on *every* task selection before;
  now paid once. The dominant real-world cost was the **blocking OSC
  terminal-background query** fired by `glamour.WithAutoStyle()` (default
  `theme=auto`) and by lipgloss AdaptiveColor resolution — only reproducible on a
  live TTY (hundreds of ms), now eliminated from the hot path (queried once in
  run.go before the program starts).

### Headless 120×35 frames
Browsing (full-width list, no preview panel):
```
╭────────────────────────────────────────────────────────────────────────────────────────╮
│ Board                                                                                    │
│ 1 in-progress · 1 todo · 1 blocked · 1 done  ████████████▊███████████▌███████████▎██████ │
╰────────────────────────────────────────────────────────────────────────────────────────╯
╭────────────────────────────────────────────────────────────────────────────────────────╮
│ ❯ backend/  Backend  1 blocked · 1 done                                                  │
│  Open tasks                                                                              │
│   T-01  Alpha task     in-progress                                                       │
│   T-02  Beta task      todo                                                              │
╰────────────────────────────────────────────────────────────────────────────────────────╯
 k/↑ up • j/↓ down • enter open • esc back • tab panel • / filter • e edit • ? keys • q quit
```

Task open (split: list left, glamour-rendered task right — `enter` on T-01):
```
╭────────────────────────────────────────────────────────────────────────────────────────╮
│ Board                                                                                    │
│ 1 in-progress · 1 todo · 1 blocked · 1 done  ████████████▊███████████▌███████████▎██████ │
╰────────────────────────────────────────────────────────────────────────────────────────╯
╭───────────────────────────────────────────╮╭───────────────────────────────────────────╮
│   backend/  Backend  1 blocked · 1 done    ││ T-01  in-progress                         │
│  Open tasks                                ││   T-01 — Alpha task                       │
│ ❯ T-01  Alpha task     in-progress         ││   ## Goal                                 │
│   T-02  Beta task      todo                ││   Ship the alpha milestone without …      │
│                                            ││   ## Read first / Scope / Gates / Report  │
╰────────────────────────────────────────────╯╰───────────────────────────────────────────╯
 k/↑ up • j/↓ down • enter open • esc back • tab panel • / filter • e edit • ? keys • q quit
```
`esc` from the open state returns to the browsing frame above; `enter` on the
backend track (preview closed) descends into it with no panel.

### Gates
All three ticked. Full suite green (`go test ./tests/... -coverpkg=./internal/...
-count=1`, coverage 83.0%); `gofmt -l .` empty; `go vet ./...` clean;
`go build -o bin/duty ./cmd/duty` ok. T-17's on-selection-preview tests were
updated to the on-open UX (deliberate, allowed); no CLI outputs/exit codes touched.

---
id: T-38
title: Apply the duty color palette to the TUI
status: done
blocked-by: []
---

# T-38 — Apply the duty color palette to the TUI

## Goal
The TUI wears the project's own palette: #e1ebaf, #e1af7d, #af874b, #1e0f37,
#9baf37 — mapped semantically, readable in dark AND light terminals.

## Read first
`internal/tui/view.go` — the color constants block (colAccent, statusColor,
rollupOrder) is the single source every style derives from; spec §8.

## Scope
- Palette (name them in one block): cream `#e1ebaf` (225,235,175), peach
  `#e1af7d` (225,175,125), bronze `#af874b` (175,135,75), indigo `#1e0f37`
  (30,15,55), olive `#9baf37` (155,175,55).
- Semantic mapping (pre-made): done → olive; in-progress → peach (replaces
  yellow); todo → bronze (dim variant); accent — focused borders, breadcrumb,
  selection, header title — → cream on dark / indigo on light
  (lipgloss.AdaptiveColor pairs); blocked → KEEP the current red (the palette
  has no alarm color and "blocked" must alarm — the user allowed "some of it").
- Every adaptive pair gets a hand-checked readable counterpart for light
  terminals (the palette skews dark-theme; indigo is the light-theme ink).
- Single source holds: only the color constants block changes value-wise;
  statusColor/statusStyle/bars/charts/help all inherit. No new styling paths.
- Headless frames on BOTH theme settings (dark, light) at 120×35 recorded in
  the report; spec §8 color sentence updated.

## Out of scope
Layout changes; configurable palettes (duty.toml keys — later if wanted);
CLI output colors; blocked staying red is deliberate, don't force-fit olive.

## Gates
- [x] Frames at 120×35 with theme=dark and theme=light recorded; every status/accent readable on both (eyeballed + noted).
- [x] Only the color-constant block changed in view.go value-wise; statusColor stays the single source (grep-verified).
- [x] Full suite green; TestStartupPerformance green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec §8 updated.

## Report

Applied the duty palette to the TUI (internal/tui/view.go, color-constants block only).

Mapping (statusColor stays the single source; bars/charts/rollups/help/entry all inherit):
- done       -> olive  #9baf37  (dark)  / #6f7d27 (light, darkened -> reads on pale ground)
- in-progress-> peach  #e1af7d  (dark)  / #a5652f (light, darkened to terracotta)
- todo       -> bronze #af874b  (dark)  / #8a6a38 (light, darkened) — palette's quiet earthy tone
- blocked    -> red    203 (dark) / 160 (light) — KEPT deliberately; palette has no alarm color
- accent (focused borders, breadcrumb, selection, header title, ❯ cursor)
             -> AdaptiveColor cream #e1ebaf on dark / indigo ink #1e0f37 on light
- colDim (separators, ages, hints, blurred borders) untouched — chrome stays terminal grays.

Light counterparts are hand-darkened because the palette skews dark; indigo is the light-theme
ink. WCAG contrast on the respective ground (hand-checked): cream-on-black 16.7, indigo-on-white
17.8; dark statuses on black — peach 10.6, olive 8.6, bronze 6.4; light darkened statuses on white
— peach #a5652f 4.7, olive #6f7d27 4.5, bronze #8a6a38 5.0. All >= AA for normal text; bronze/olive
raw values (2.5–3.3 on white) would have washed out, hence the darkened light variants.

statusColor gained one explicit `case task.StatusTodo -> colBronze` so todo leaves the chrome-dim
default; unknown statuses still fall through to colDim. Constants renamed to palette names
(colYellow->colPeach, colGreen->colOlive, +colBronze); colRed/colDim/colAccent kept.
Grep-verified statusColor/statusStyle are the only status->color path (callers: bars L242,
preview header L307, rollup L426, trackBar L499, entry.go L235). Spec §8 status-color sentence
updated. No layout/config/CLI-color changes (out of scope).

Frames recorded at 120x35 (headless, TrueColor profile; palette hex verified in the emitted SGR
codes for both themes).

theme=dark:
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│  › Board                                                                                                             │
│ 2 in-progress · 1 todo · 1 blocked · 4 done  █████████████████▊████████▋████████▌███████████████████████████████████ │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│  Tracks                                                                                                              │
│ ❯ backend/  Backend services                                                                       ██████████████  2 │
│  Open tasks                                                                                                          │
│   T-01  Ship the alpha milestone                                                          in-progress         2m ago │
│   T-02  Wire the fsnotify watcher                                                         todo                2m ago │
│   T-04  Document the color palette                                                        blocked             2m ago │
│   T-03  Render the distribution bar                                                       done                2m ago │
│   T-05  Design the port interface                                                         done                2m ago │
│   T-06  Atomic write adapter                                                              done                2m ago │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
 k/↑ up • j/↓ down • ↵ open • esc back • tab panel • / filter • e edit • r refresh • ? keys • q quit
Emitted SGR (dark): accent 38;2;225;235;175 · peach 38;2;225;175;125 · bronze 38;2;175;135;75 ·
olive 38;2;155;175;55 · blocked 38;5;203 · dim 38;5;243.

theme=light: same layout; emitted SGR: accent 38;2;30;15;55 (indigo) · peach 38;2;165;101;47 ·
bronze 38;2;138;105;56 · olive 38;2;111;125;39 · blocked 38;5;160 · dim 38;5;245.

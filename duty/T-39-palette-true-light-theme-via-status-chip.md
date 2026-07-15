---
id: T-39
title: Palette-true light theme via status chips
status: done
blocked-by: []
---

# T-39 — Palette-true light theme via status chips

## Goal
Light terminals show the palette EXACTLY as given: status words become chips —
palette hue as background, indigo ink on top — and bars use the raw hues.
Dark theme stays as T-38 shipped it (approved).

## Read first
`internal/tui/view.go` palette block + `statusColor`/`statusStyle` (T-38 +
its simplify addendum); why fg-only fails on light: cream 1.2:1, peach 1.9,
olive 2.3 on white — only indigo (17.8) and marginally bronze (3.0) read.

## Scope
- Dark theme: UNCHANGED (user-approved).
- Light theme, statuses as chips: `statusStyle` becomes theme-aware — dark →
  Foreground(statusColor) as today; light → Background(raw palette hue) +
  indigo `#1e0f37` foreground, one space of horizontal padding: done=olive bg,
  in-progress=peach bg, todo=cream bg, blocked=red bg + white fg (alarm
  stays). The hand-darkened light foreground variants from T-38 are deleted.
- Bars/charts/rollup counts on light: `statusColor` returns the RAW palette
  hues (area fills and the chips carry the color; no fg-contrast dependency).
  Rollup count text on light: indigo, with a thin chip per count or plain
  indigo numbers — implementer picks the cleaner frame, records it.
- Single source holds, restated: `statusColor` + `statusStyle` remain the ONLY
  status→visual paths (grep gate stays).
- Measured contrast for every light combination (indigo on cream/peach/olive/
  bronze; white on red) in the report; 120×35 light AND dark frames recorded —
  dark frame byte-identical to pre-change.
- Spec §8 color sentence updated.

## Out of scope
Dark theme changes; layout; config keys; CLI colors; new palette values
(the five given hexes + indigo-ink + alarm red + chrome dim are the whole
vocabulary).

## Gates
- [x] Dark 120×35 frame byte-identical to pre-change; light frame shows palette-true chips — both recorded with contrast numbers.
- [x] statusColor/statusStyle still the only status→visual paths (grep); T-38's darkened light variants deleted.
- [x] just check green (fmt/vet/lint/tests incl. TestStartupPerformance); build ok; spec §8 updated.

## Report

Palette-true light theme landed. On light terminals statuses now render as chips
(raw palette hue as background, indigo ink) and bars/area fills carry the raw
hues; the dark theme is untouched (proven byte-identical below). T-38's
hand-darkened light foreground variants (#a5652f/#8a6a38/#6f7d27) are deleted.

### Files changed
- `internal/tui/view.go` — palette block rewritten (raw hues + `colInk`
  indigo + `colChipText` white + `colTodo` bronze/cream adaptive; darkened
  variants removed); `statusStyle` now theme-aware (dark = `Foreground(statusColor)`
  unchanged; light = `Background(statusColor)` + indigo ink + `Padding(0,1)`,
  blocked = red bg + white ink); `statusColor` returns raw palette hues on light
  (`colTodo` = cream on light / bronze on dark); `trackBar` fills with
  `Foreground(statusColor)` directly (a chip would break the bar).
- `task-system-spec.md` §8 — status/chip sentence updated.

### Single source held
`statusColor` + `statusStyle` remain the only status→visual mappers. Callers:
`statusStyle` at entry.go:235 (task row), view.go:323 (preview header), view.go:442
(rollup); `statusColor` at view.go:258 (ntcharts bar), view.go:515 (trackBar).
No status→color switch or palette literal lives outside view.go's two functions.
Rollup counts on light: kept as chips (statusStyle) — the cleaner frame, and it
preserves the single source (plain-indigo numbers would need a second path).

### Measured contrast (light, on white where noted)
Chips (indigo ink #1e0f37 on raw hue):
| status | chip bg | contrast |
|---|---|---|
| todo | cream #e1ebaf | 14.2:1 |
| in-progress | peach #e1af7d | 9.0:1 |
| done | olive #9baf37 | 7.3:1 |
| blocked | red (ANSI 160 ≈ #d70000), white ink | 5.4:1 |

Reference (indigo on bronze #af874b = 5.4:1) documents why todo's chip shifts to
cream. Rejected fg-only-on-white (why chips): cream 1.2:1, peach 1.9:1, olive
2.3:1 — unreadable as ink; bronze 3.0:1, indigo 17.8:1.

### Dark-frame identity proof
Rendered the 120×35 browsing frame with `lipgloss.SetColorProfile(TrueColor)` +
`SetHasDarkBackground(true)` on the pre-change view.go (git stash) and again after
the edits. `cmp` reports the two dark frames byte-identical (color codes included:
peach 225;175;125, bronze 175;135;75, olive 155;175;55 — unchanged). The light
frame's distinct chip backgrounds are 48;2;225;235;175 (cream), 48;2;225;175;125
(peach), 48;2;155;175;55 (olive), 48;5;160 (blocked red); inks 38;2;30;15;55
(indigo) and 38;2;255;255;255 (white, blocked).

### Frames (120×35, ANSI stripped; body trimmed to the rows)
Light — status words are chips (padded cells carry the hue bg in a real terminal;
the 2-cell chip padding is why the dim age truncates to `just …`, layout §8 kept):
```
╭────────────────────────────────────────────────────────────────────────────────╮
│ Board                                                                            │
│  2 in-progress  ·  2 todo  ·  1 blocked  ·  1 done   ████████████████▌██████████ │
╰────────────────────────────────────────────────────────────────────────────────╯
│  Tracks                                                                          │
│ ❯ backend/  Backend                                                ████████  2   │
│  Open tasks                                                                      │
│   T-02  Beta task                              in-progress          just …       │
│   T-03  Gamma task                             todo                 just …       │
│   T-04  Delta task                             blocked              just …       │
│   T-01  Alpha task                             done                 just …       │
```
Dark — unchanged from T-38 (status word in its palette hue as foreground):
```
│ 2 in-progress · 2 todo · 1 blocked · 1 done  ██████████▋██████████▍████▏████████ │
│   T-02  Beta task                              in-progress         just now       │
│   T-03  Gamma task                             todo                just now       │
│   T-04  Delta task                             blocked             just now       │
│   T-01  Alpha task                             done                just now       │
```

### Gates
`just check` green: go vet clean, golangci-lint 0 issues, `go test ./tests/...`
ok at 86.5% coverage (TestStartupPerformance and all frame tests pass; they assert
structure/substrings, not color literals). Build ok. All three gates ticked.

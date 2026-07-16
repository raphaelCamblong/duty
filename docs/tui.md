# TUI

`duty tui` opens a live, read-only view of your board. It never writes — edits
go through the CLI or your editor — and it re-reads the tree the moment a file
changes, so your agent's progress shows up as it happens. Startup is instant:
nothing renders a task until you open one.

![The duty board, dark theme](/screens/board-dark.png)

## What you see

- **Header** — the track path as a breadcrumb (each segment jumps to that
  ancestor), the subtree's per-status counts, and a one-line distribution bar.
- **List** — sub-tracks first, under a "Tracks" header, each with a
  right-aligned bar of its subtree; then tasks under their section headers,
  each showing id, title, colored status (in-progress gets a small animated
  spinner while work is live), gates `2/3`, and a relative age. A
  task with unmet dependencies shows a dim `waits T-01,T-03` beside its status;
  a `⚠` marks a row whose board status disagrees with the file. Rows group by
  status by default (in-progress first); the age column always shows, the gate
  column hides below 100 columns.
- **Preview** — opens on the right when you open a task. Below ~80 columns it
  takes over full-screen.

## Moving around

`j`/`k` (or the arrows) move; `enter` opens a task or descends into a track;
`esc` steps back — closing the preview, clearing a filter, or going up a track,
in that order. `tab` switches panel focus. The mouse works too: rows, panels,
and breadcrumb segments are click zones, double-click opens, and the wheel
scrolls whatever's under it.

## Opening a task

`enter` on a task splits the view and renders the file on the right, focus on
the preview; `esc` closes it. The preview is topped by a pinned header — id,
status, gates, track, age, and any `blocked-by` ids (met ones struck through).
`enter` on a track descends into it instead; with a preview already open it
shows the track's summary card.

![A task open in the split preview](/screens/task-preview.png)

## Filtering

`/` opens a fuzzy filter. Matches rank the rows while it's active, and the
status grouping steps aside; `esc` clears it. A filter with no matches shows the
list's empty line.

![The fuzzy filter narrowing the list](/screens/filter.png)

## Keys

| Key | Does |
|---|---|
| `j` / `k` | down / up (arrows work too) |
| `enter` | open a task / descend a track |
| `esc` | back: close preview, clear filter, up a track |
| `tab` | switch panel focus |
| `/` | fuzzy filter |
| `e` | open the task in your editor (suspends, then resumes) |
| `t` | toggle the age column (on by default) |
| `s` | toggle raw board order (grouped by default) |
| `r` | re-scan now |
| `?` | toggle the full key grid |
| `q` | quit |

## Theming

Both themes ship tuned: the raw duty palette on dark terminals, AA-readable inks
on light ones. `theme` in your config picks the background mode, and
`[tui.palette]` recolors any slot — see [Config](/config/).

![The same board on a light terminal](/screens/board-light.png)

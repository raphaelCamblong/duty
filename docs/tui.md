# TUI

`duty tui` — a read-only live board. **Per frame:** scan the tree for boards, parse
each `BOARD.md` for sections + row order, parse every task's frontmatter and count its
gates. Files win; a board/file mismatch renders a `⚠` badge on the row.

**Layout — browse full-width, preview on open** (all styling lipgloss). Startup is
instant: no markdown renderer is built and no terminal query fires until a task is *opened*.

- **Header:** breadcrumb of the track path — each board's H1 title (see tracks.md),
  each segment a clickable zone jumping to that ancestor — plus the **subtree**'s
  per-status counts in status colors and a one-line status-distribution bar.
- **Left panel** (full width browsing; ~38%, min 30 cols, with a preview open): a
  `bubbles/list`, sub-tracks first under a non-selectable **"Tracks" header** (skipped
  by navigation, hidden while filtering) — name and title left, then a **right-aligned
  fixed-width status-distribution bar** of its subtree flush at the line end with a
  dim total, the bar column at the same x on every row
  (`backend/  Title      ▰▰▰▰▰▰▱▱  7`): proportional colored segments, every non-zero
  status ≥1 cell, a dim `empty` when taskless; the title ellipsis-truncates first, the
  bar is never dropped. Then tasks under their section headers: id, title, colored
  status word, gates `2/3`, dim relative age, drift badge. The **age column is always
  shown** (`t` toggles); the **gate column hides below 100 columns**. Rows within a
  section are **status-grouped for display by default** — in-progress, todo, blocked,
  done, unknown last — a *stable* sort, board build order as tiebreak, presentation
  only; `s` toggles raw board order (session-only, no config key). `/` opens the fuzzy
  filter — match rank orders rows while active, the status sort steps aside. Empty
  boards show a centered dim hint; a no-match filter shows the list's no-items line.
- **Colors.** Dark theme (frozen): status words carry the raw duty palette as
  foreground — `todo` bronze, `in-progress` peach, `done` olive, `blocked` red —
  accents cream. Light theme: the raw hues are too pale on white, so status words are
  **flat AA-darkened inks** (no chips, no backgrounds): accent/ink navy `#1f3a5f`
  (11.5:1), in-progress blue `#3a6ea5` (5.3:1), todo amber `#8a6d00` (4.9:1), done
  olive `#6f7d27` (4.5:1), blocked red (5.4:1), dim grey (5.25:1), body black. Bars
  fill with the raw hues on both themes. The **selected row is bold across the whole
  line**, both themes. This palette is the default; every slot is overridable from
  `[tui.palette]` (see config.md).
- **Right panel — on open only:** `enter`/double-click on a task opens the split — the
  file rendered by glamour in a viewport, focus on the preview; `esc` closes. `enter`
  on a track descends; with a preview open it shows the track's summary card (title,
  per-status counts, bar, sections with row counts, subtree drift count). A task
  preview is topped by a pinned header — `id · status (colored) · gates n/m · track
  title · age` (age dim, trailing last so narrow truncation drops it before the
  `blocked-by` ids and drift flag). The preview reads from the rows' scan snapshot
  (zero extra I/O); one glamour renderer, built lazily on first open, rebuilt only on
  width change; content re-renders on re-scan.
- **Footer:** key-hint bar (`?` toggles the full grid). **Responsive:** below ~80
  columns an open preview takes over the body full-screen.

**Keys:** `j/k` move, `enter` open/descend, `esc` back (close preview / clear filter /
up one track), `tab` panel focus, `/` filter, `e` open in `$EDITOR` (suspend, resume),
`t` age column (default on), `s` sort toggle (default grouped), `r` manual re-scan,
`?` help, `q` quit. **Mouse:** panels, rows, and breadcrumb segments are hit-zones —
click selects/focuses/jumps, double-click opens/descends, the wheel scrolls the
hovered panel; preview scrolling is spring-smoothed, never slower than the keyboard.

**Live refresh:** fsnotify watches every directory (per-directory, not recursive; a
directory event re-walks so new tracks appear live). Debounce ~100 ms, then **re-scan
everything** — a full re-read beats any cache and keeps the TUI stateless. No polling,
no IPC.

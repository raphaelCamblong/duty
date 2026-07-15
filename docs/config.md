# Config

TOML, read-only, merged over built-in defaults: **user**
(`os.UserConfigDir()/duty/config.toml`) < **project** (`duty.toml` next to the root
`BOARD.md`; its presence also marks the tree root explicitly — otherwise the walk-up
stops at the topmost `BOARD.md`). Missing files are fine. Only the root `duty.toml` is
read; one inside a track is an error (a second root).

```toml
editor = "nvim"        # falls back to $EDITOR, then vi

[tui]
theme = "auto"         # auto | dark | light — background mode, not colors

[tui.palette]          # optional per-slot color overrides; unset = the default
accent = "#e1ebaf"     # a bare string sets both the light and dark channel
todo = { light = "#8a6d00", dark = "#af874b" }   # or set each channel apart
```

`[tui.palette]` recolors the TUI's semantic slots — `accent`, `dim`, `todo`,
`in-progress`, `done`, `blocked` — over the frozen default described in tui.md; an
omitted slot (or, in the table form, an omitted channel) keeps its default, so the
zero config renders the default palette byte-for-byte. Each value is a `#rrggbb` hex
triplet or an ansi index `0-255`; a malformed value is an error naming the key
(`tui.palette.todo.dark`). The palette is a distinct table from the `theme` mode
selector above (TOML forbids one key being both a string and a table). Status colors
ink the word directly and their dark hue fills that status's distribution bar, so one
slot drives both.

Keys get added when a hardcoded value hurts, not before. Config tunes presentation
only — statuses, naming, and board structure are the convention, never settings.

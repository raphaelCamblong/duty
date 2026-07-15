# Config

duty runs with zero configuration — every setting has a default. When you want
to change one, it reads TOML from two optional places, project winning over
user:

- **User**, per machine: `os.UserConfigDir()/duty/config.toml`.
- **Project**, committed with the tree: `duty.toml` beside the root `BOARD.md`.

The project file's mere presence also pins the tree root. Only that root
`duty.toml` is read — a second one inside a track is an error.

## A config file

```toml
# Editor for the TUI's `e` key. Falls back to $EDITOR, then vi.
editor = "nvim"

[tui]
# Background mode, not colors: auto follows the terminal.
theme = "auto"        # auto | dark | light

# Recolor any of the TUI's six semantic slots. Unset = the frozen default.
[tui.palette]
accent = "#1f3a5f"                               # one value sets both channels
todo   = { light = "#8a6d00", dark = "#af874b" } # or split light and dark
```

That's every key. The palette slots are `accent`, `dim`, `todo`,
`in-progress`, `done`, and `blocked`; each value is a `#rrggbb` hex triplet or
an ANSI index `0-255`. A bare string colors both channels; the table form sets
light and dark apart. An omitted slot — or an omitted channel — keeps its
default, so the empty file renders the default palette exactly.

:::tip[Light or dark]
`theme` only picks the background mode. The status hues are tuned twice — the
raw palette on dark terminals, AA-readable inks on light ones — so both read
well without touching `[tui.palette]`.
:::

Config tunes presentation only. Statuses, naming, and board structure are the
convention, never settings — a key that changed a file format would be a bug.

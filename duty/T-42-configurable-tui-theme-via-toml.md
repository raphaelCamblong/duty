---
id: T-42
title: Configurable TUI theme via TOML
status: todo
blocked-by: []
---

# T-42 — Configurable TUI theme via TOML

## Goal
The TUI palette is editable from `config.toml` under `[tui.theme]`, with the
current palette as the default. The colors — already centralized in one block
in `internal/tui/view.go` — become a proper `Theme` value threaded through the
model instead of package-level vars.

## Read first
`internal/tui/view.go` color block (already the SINGLE source — no scattered
literals to hunt); `internal/config/config.go` (TUI settings, infra layer —
keep it lipgloss-free); `docs/spec.md` §7 (config) + §8 (palette); CLAUDE.md
(no package-level mutable state; config tunes presentation only).

## Scope
- New `internal/tui/theme.go`: a `Theme` struct with semantic slots (accent,
  dim, todo, in-progress, done, blocked) as `lipgloss.AdaptiveColor`.
  `DefaultTheme()` returns the current palette verbatim. Likely simplification
  (verify, don't assume): today's bar hues peach/bronze/olive equal the `.Dark`
  of the todo/in-progress/done inks, so one adaptive slot per status can feed
  BOTH the adaptive text color and the bar fill (bar = slot `.Dark`). The
  byte-identical gate is the real guarantee — match current output exactly.
- Replace the package-level `col*` vars: store `theme Theme` on the Model
  (built in `New` from config), thread it to the render helpers (statusColor /
  statusStyle / trackBar / header / etc. become Model methods or take Theme).
  No package-level mutable state remains.
- Config surface: `[tui.theme]` table, one entry per slot, each
  `{ light = "...", dark = "..." }` (a bare string sets both). Parsed as
  strings in `internal/config` (NO lipgloss import in infra); tui overlays them
  onto `DefaultTheme()`, an unset slot keeping the default. Validate each value
  (hex `#rrggbb` or ansi `0-255`) — malformed → a clear error naming the key.
- `docs/spec.md` §7 documents the `[tui.theme]` keys + "default = current
  palette"; §8 notes the palette is themeable.

## Out of scope
New colors or slots beyond the existing set; per-widget theming; runtime theme
switching in the TUI; the auto/dark/light selector (`theme` key stays as-is);
writing a sample config from `init`.

## Gates
- [ ] Default (no `[tui.theme]`) renders 120x35 dark AND light frames
  byte-identical to current HEAD (cmp-proven, like the T-39/T-40 color tasks).
- [ ] A `[tui.theme]` override in a test config recolors exactly that slot
  (text AND bar), other slots unchanged; a malformed color errors naming the key.
- [ ] No package-level color vars remain (grep); the Theme is threaded via the
  Model, no package-level mutable state.
- [ ] `just check` green; build ok; `docs/spec.md` §7 + §8 updated.

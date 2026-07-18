---
id: T-42
title: Configurable TUI theme via TOML
status: done
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
- [x] Default (no `[tui.theme]`) renders 120x35 dark AND light frames
  byte-identical to current HEAD (cmp-proven, like the T-39/T-40 color tasks).
- [x] A `[tui.theme]` override in a test config recolors exactly that slot
  (text AND bar), other slots unchanged; a malformed color errors naming the key.
- [x] No package-level color vars remain (grep); the Theme is threaded via the
  Model, no package-level mutable state.
- [x] `just check` green; build ok; `docs/spec.md` §7 + §8 updated.

## Report

## Report

Configurable TUI palette shipped. The centralized color block in
`internal/tui/view.go` is now a `Theme` value threaded through the model and
the list delegate — no package-level color vars remain.

**Files**
- `internal/tui/theme.go` (new): `Theme` struct (6 adaptive slots: accent, dim,
  todo, in-progress, done, blocked) + `DefaultTheme()` (frozen §8 palette
  verbatim) + `themeFromConfig` overlay/validator + the style/status methods
  (statusStyle, statusColor, accent, dim, section, crumb, alert, focusBox,
  blurBox, panelBox, cursorMark).
- `internal/tui/view.go`: removed the `col*`/`*Style` var block and the
  `statusStyle`/`statusColor`/`panelBox`/`cursorMark` free funcs; the render
  helpers (stateLine, statusBar, barData, statusRollup, rollupOrEmpty, trackBar,
  trackBarCell, taskHeader) are now `Theme` methods; Model methods call
  `m.theme.*`.
- `internal/tui/model.go`: `theme` field split into `theme Theme` (palette) and
  `mode string` (dark/light for glamour); `New` builds the theme from
  `cfg.TUI.Palette` and returns a malformed-color error; `newList`/`newDelegate`
  take the theme.
- `internal/tui/entry.go`: delegate carries `theme Theme`.
- `internal/config/config.go`: `[tui.palette]` table parsed as strings
  (`Palette` + `Color` with `UnmarshalTOML` accepting a bare string or an inline
  `{ light, dark }` table) — no lipgloss import; the TUI validates and overlays.
- `docs/spec.md` §7 (palette keys + default) and §8 (palette themeable) updated.

**Byte-identity proof.** `tests/tui_theme_test.go` captures the 120x35 dark AND
light browsing frames of the four-status fixture from HEAD (TrueColor forced,
age column hidden for determinism) into `tests/testdata/tui_frame_{dark,light}.golden`.
After the refactor `TestThemeDefaultByteIdentity` compares the default-theme
render against those goldens — both pass, so zero-config output is byte-for-byte
HEAD. Frames carry real ANSI (peach 225;175;125 on dark, blue 58;110;165 ink on
light, etc.), so the comparison is meaningful, not color-stripped.

**Override + validation tests.** `TestThemePaletteOverride` overrides the `done`
slot (bare-string and dark-channel table forms) and asserts the olive hue
(155;175;55, done ink AND bar) is fully replaced by the override while the
in-progress peach is untouched. `TestThemeMalformedColor` feeds bad hex, an
out-of-range ansi index, and a color name; each errors naming the key
(`tui.palette.blocked.dark`/`.light`). `TestLoadPalette` proves the TOML surface
(bare string + inline table, coexisting with `theme = "dark"`, unset slots nil).

**Deviation (spec bug fixed in-change).** The task named the table `[tui.theme]`,
but TOML forbids `theme` being both the existing `theme = "auto"` string and a
table (verified: "Key 'tui.theme' has already been defined"). Used `[tui.palette]`
instead — the mode selector `theme` stays as-is — and documented it in §7.

**Simplification confirmed.** peach/bronze/olive == the `.Dark` of the
in-progress/todo/done ink slots, so one slot feeds both the adaptive text and the
bar fill (bar = slot `.Dark`). blocked and the unknown fallback keep the full
adaptive color for the bar (their .Light≠.Dark), which the byte-identity gate
required.

**Gates:** `gofumpt -l .` empty, `go vet ./...` clean, `golangci-lint` 0 issues,
full suite green (87.3% coverage), build ok.

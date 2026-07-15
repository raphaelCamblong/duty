---
id: T-38
title: Apply the duty color palette to the TUI
status: todo
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
- [ ] Frames at 120×35 with theme=dark and theme=light recorded; every status/accent readable on both (eyeballed + noted).
- [ ] Only the color-constant block changed in view.go value-wise; statusColor stays the single source (grep-verified).
- [ ] Full suite green; TestStartupPerformance green; golangci-lint 0 issues; gofumpt -l . empty; go vet clean; build ok; spec §8 updated.

## Report

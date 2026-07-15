---
id: T-40
title: Light theme flat colors, bold selection, small-screen priority
status: todo
blocked-by: []
---

# T-40 — Light theme flat colors, bold selection, small-screen priority

## Goal
Light theme drops the chips for flat colored text in the screenshot's
vocabulary (blues, amber, grey, black); the selected row is unmistakably bold;
small screens always show the age and hide gates instead.

## Read first
`internal/tui/view.go` (statusStyle chips from T-39 — being replaced on light;
statusColor; selection styles in entry.go titleStyle/selStyle), T-31's width
threshold logic for the age column; user feedback: chips rejected, dark theme
still approved and frozen.

## Scope
1. Light theme, flat text (NO chips — delete the chip styling): from the
   screenshot vocabulary, pre-made mapping (all AA-readable on white,
   measure and record):
   - accent/ink (borders, breadcrumb, selection, ids, header title):
     darker blue — navy family (e.g. #1f3a5f, adjust for AA)
   - todo: dark amber from the yellow family (e.g. #8a6d00)
   - in-progress: medium blue from the light-blue family (e.g. #3a6ea5)
   - done: dark olive #6f7d27 (brand continuity with the bars)
   - blocked: red (alarm, unchanged); dim/chrome: medium grey; body: black.
   Bars keep raw palette hues on both themes (area fills). Dark theme stays
   byte-identical (prove with the pre/post frame cmp like T-39 did).
2. Selected row: BOLD across the whole focused line (both themes) — the
   chevron alone is not enough feedback. Italic rejected (spotty terminal
   support).
3. Small screens: the age column is ALWAYS shown (still toggleable with t);
   the gates column hides below the narrow threshold instead (~100 cols).
   Preview header keeps both when open.
4. Spec §8 updated (colors sentence, selection emphasis, narrow-screen
   column priority).

## Out of scope
Dark theme changes (frozen); layout beyond the column swap; config keys;
CLI colors.

## Gates
- [ ] Dark 120×35 frame byte-identical pre/post (cmp-proven); light 120×35 frame flat-colored, no chip backgrounds in SGR output, contrast numbers recorded.
- [ ] Selected row renders bold in both themes (frame-verified); 80×24 frame shows age column and NO gates column; 120×35 shows both.
- [ ] just check green incl. TestStartupPerformance; build ok; statusColor/statusStyle still the only status→visual paths; spec §8 updated.

## Report

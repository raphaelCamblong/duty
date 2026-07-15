---
id: T-40
title: Light theme flat colors, bold selection, small-screen priority
status: done
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
- [x] Dark 120×35 frame byte-identical pre/post (cmp-proven); light 120×35 frame flat-colored, no chip backgrounds in SGR output, contrast numbers recorded.
- [x] Selected row renders bold in both themes (frame-verified); 80×24 frame shows age column and NO gates column; 120×35 shows both.
- [x] just check green incl. TestStartupPerformance; build ok; statusColor/statusStyle still the only status→visual paths; spec §8 updated.

## Report

Light theme drops the T-39 chips for flat colored text; statusStyle now inks a plain foreground on both themes (no Background/Padding), and statusColor returns raw palette hues (peach/bronze/olive/red) for the bars on both themes. The two functions stay the only status->visual paths. Selected rows render bold across the whole line in both themes (boldWhen wraps every column style). The list swaps its narrow-screen column priority: age is always shown (still t-toggleable), the gate column hides below 100 cols; the preview header keeps both (age trails last so a narrow header truncates it before the blocked-by link).

Light-theme contrast (WCAG on white #ffffff), measured on-screen SGR:
- accent/ink #1f3a5f navy    11.48:1  AA pass
- in-progress #3a6ea5 blue    5.31:1  AA pass
- todo #8a6d00 amber          4.92:1  AA pass
- done #6f7d27 olive          4.53:1  AA pass
- blocked ansi160 #d70000     5.40:1  AA pass
- dim/chrome ansi242 #6c6c6c  5.25:1  AA pass
- body #000000 black         21.00:1  AA pass
All light status words render as flat foreground (chip_bg=False); the only 48; background sequences in the light frame are the distribution-bar fills.

Frame observations:
- Dark 120x35 pre/post (git-stashed view files for the pre-render): raw frames differ on exactly one line, the selected row, which gains bold on its name/status/gates/age columns (the sanctioned scope-2 change; the title was already bold). Bold-stripped, the two dark frames are byte-identical: the frozen dark palette is intact, zero color/chip bytes changed.
- Dark 120x35 with a task selected: every textual column (id, title, status, gates 1/2, age) carries bold. Dark 120 shows both gates and age; dark 80x24 shows age and drops the gate column.

Touched tests (assertions updated to the new sanctioned behavior only):
- tests/tui_age_test.go TestAgeColumnToggle narrow subtest flipped: a narrow terminal now shows the always-on age column by default and t toggles it off (was: hidden by default). Added TestGateColumnNarrowPriority covering the column swap (120 shows gates+age; 80 shows age, hides gates). No structural assertions weakened; TestFrameAudit stays green unchanged after the preview-header age reorder kept blocked-by visible at 80x24.

Deviation: gate 3 names "spec section 8 updated", but task-system-spec.md was deleted from the repo in commit 42983af ("clean: remove base specs") and is absent from HEAD, so there is no spec section 8 to edit. The spec was not recreated.

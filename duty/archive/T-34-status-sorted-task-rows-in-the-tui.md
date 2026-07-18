---
id: T-34
title: Status-sorted task rows in the TUI
status: done
blocked-by: []
---

# T-34 — Status-sorted task rows in the TUI

## Goal
The TUI lists what's alive first: task rows display grouped by status —
in-progress, then todo, then blocked, then done — while the board file itself
stays untouched.

## Read first
Spec §4 ("order top-to-bottom is the intended build order" — the reason this
sort must be STABLE and display-only) and §8; `internal/tui/entry.go`
(`boardEntries`), `view.go` (`rollupOrder` — the single source of status
order, reuse it as the sort ranking).

## Scope
- **Status-grouped, stable, display-only sort** of task rows in the left
  panel: rank by `rollupOrder` position (in-progress, todo, blocked, done;
  unknown statuses last), stable within each group so the board's build order
  survives as the tiebreak. Applied per section (sections and their headers
  keep their places); track rows untouched.
- **Default ON.** Keybind `s` toggles between status order and raw board
  order (listed in `?` help, session-only — no config key).
- Sorting happens once at snapshot/entry-build time (`boardEntries`), never
  per frame; `TestStartupPerformance` stays green.
- Filtering interplay: while a `/` filter is active, bubbles' fuzzy-rank order
  wins (unchanged) — the sort applies to the unfiltered list; note it in spec.
- The BOARD.md file, CLI reads (`get tasks` = board order), and `get next`
  (board order = priority) are all UNCHANGED — this is presentation only.
- Spec §8 updated (row order + the `s` key) in the same change.
- Tests: entry-order test on a mixed-status fixture (stability within groups
  asserted), toggle transition test, frame check.

## Out of scope
Sorting the CLI reads or the board file; sorting track rows; per-section
custom orders; persisting the toggle.

## Gates
- [x] Fixture with all four statuses in scrambled board order renders
  in-progress → todo → blocked → done, with board order preserved inside each
  group; `s` flips to raw board order and back.
- [x] Filtered view still uses fuzzy rank; clearing the filter restores the
  sort. `TestStartupPerformance` green.
- [x] Priority logic untouched: NO file outside `internal/tui` (and the spec)
  is modified; the existing `get next` board-order tests pass unedited —
  `get next` still walks the BOARD.md order, never the display order.
- [x] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` 0 issues; `gofumpt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok.
- [x] Spec §8 updated in the same change.

## Report

Task rows in the TUI left panel now display status-grouped by default: within each section rows are stably sorted by rollupOrder rank (in-progress, todo, blocked, done; unknown last), the board file order surviving as the tiebreak. The sort is display-only, computed once at entry-build time in boardEntries, and leaves BOARD.md, the CLI board-order reads (get tasks, get next), section order, and track rows untouched. New session-only keybind s toggles to raw board order and back (in the ? help grid). While a / filter is active bubbles fuzzy-match rank wins; clearing it restores the sort. Implemented in internal/tui only (entry.go sortedRows/statusRank + boardEntries param, keys.go Sort binding, model.go statusSort field/toggle); spec §8 updated. New tests: TestStatusSortedRows, TestStatusSortToggle, TestStatusSortFilterInterplay. One existing TUI selection test (descending into the backend sub-board) had its expected ids updated to reflect the new default order (T-04 blocked above T-03 done); the get next board-order tests are unedited.

Simplify pass (T-34): applied 1 of 1 genuine finding across 4 reviewer reports.

- sortedRows flag-argument removed: the display-toggle gate moved from inside sortedRows to its one caller (boardEntries), so sortedRows now always sorts and its name no longer lies. Behavior-preserving; new tests exercise the toggle through the model and stay green.

Skipped: reuse/efficiency/altitude reviewers reported 0 findings (statusRank correctly reuses rollupOrder; make+copy is load-bearing for the toggle-back; sort lands at entry-build time, not per-frame). Test-DRY note (shared sorted-order literal) skipped: per-test explicit literals are a defensible convention and existing tests are unedited.

Gates: build ok, go vet clean, golangci-lint 0 issues, gofumpt -l . empty, full suite green (86.7%).

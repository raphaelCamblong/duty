---
id: T-34
title: Status-sorted task rows in the TUI
status: todo
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
- [ ] Fixture with all four statuses in scrambled board order renders
  in-progress → todo → blocked → done, with board order preserved inside each
  group; `s` flips to raw board order and back.
- [ ] Filtered view still uses fuzzy rank; clearing the filter restores the
  sort. `TestStartupPerformance` green.
- [ ] Full suite green (`go test ./tests/... -coverpkg=./internal/... -count=1`);
  `golangci-lint run` 0 issues; `gofumpt -l .` empty; `go vet ./...` clean;
  `go build -o bin/duty ./cmd/duty` ok.
- [ ] Spec §8 updated in the same change.

## Report

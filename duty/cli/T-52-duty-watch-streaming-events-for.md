---
id: T-52
title: "duty watch: streaming events for orchestrators"
status: done
blocked-by: []
---

# T-52 — duty watch: streaming events for orchestrators

## Goal
An orchestrating agent reacts to state changes the second they happen:
`duty watch --agent` streams one TSV line per task state change, powered by
the same watcher the TUI uses.

## Read first
`internal/tui/watch.go` (fsnotify per-dir watches, ~100ms debounce, full
re-scan — REUSE this, likely by promoting the watcher out of tui into a
shared internal package so tui and cli both consume it; respect the
dependency rule), `internal/tui/scan.go` (snapshot diffing needs two
snapshots), spec-of-record: docs/internals.md.

## Scope
- `duty watch [--agent] [--in <track>]` — long-running (the one exception to
  one-shot; document it): prints one line per detected change by diffing
  consecutive snapshots: `<RFC3339>\t<event>\t<id>\t<field>\t<old>\t<new>`
  (events: status, claimed-by, created, deleted, moved, gates). Human mode
  prints a readable line; `--agent` the TSV. Exits cleanly on SIGINT; exits
  non-zero if the tree disappears.
- Watcher promotion: extract the fsnotify layer to a shared home (e.g.
  internal/watch) consumed by both tui and the new command — one watcher
  implementation, zero duplication (grep gate).
- Initial output: nothing on start (state, not history) — orchestrators pair
  it with one `get tasks --agent` for the baseline.
- Docs: cli.md section + a "for orchestrators" note in getting-started.
- Tests: snapshot-diff unit tests (pure); an end-to-end test driving a
  scratch tree with CLI mutations while watch runs, asserting emitted lines
  (mirror the TUI's TestWatcherRefresh technique).

## Out of scope
Filtering flags beyond --in; JSON; history/replay (files+git are history);
webhooks; the TUI.

## Gates
- [x] Live test: `duty status` / `create` / `move` in another process each
  produce exactly one correct event line within the debounce window.
- [x] One watcher implementation shared by tui and watch (grep proves no
  duplicate fsnotify setup).
- [x] Ctrl-C exits 0; tree deletion exits non-zero with one lowercase line.
- [x] `just check` green; docs updated.

## Report

### 2026-07-16 13:01 — done

Promoted the fsnotify layer into internal/watch, shared by the TUI and the new duty watch command (grep proves one fsnotify home). watch [--agent] [--in] is the one long-running command: it diffs consecutive snapshots and prints one line per changed field (status, claimed-by, created, deleted, moved, gates), TSV under --agent, readable otherwise. Nothing on start; SIGINT exits 0; the tree disappearing exits non-zero with one lowercase line. Pure Diff/Snapshot units plus an end-to-end test driving CLI mutations while the watcher runs, and a command-level test for format + exit codes.

### 2026-07-16 13:11 — done

Applied two behavior-preserving simplify findings from the visibility round: sortedIDs now uses slices.Sorted(maps.Keys) for the union, and boardOrder collapses its two parallel maps (present + seen) into one. Skipped the rest as test-touching, tui-file, behavior-changing, or architectural relocations. just check green.

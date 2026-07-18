---
id: T-57
title: Explore archived tasks from the TUI
status: done
blocked-by: []
---

# T-57 — Explore archived tasks from the TUI

## Goal
The archive stops being invisible: one keybind reveals archived tasks in
place — dim, readable, previewable — and tracks emptied by archiving stay out
of sight until you ask.

## Read first
`internal/tui/scan.go` (Boards/TaskFileNames skip archive/ today),
`entry.go`/`view.go` (section headers, dim styles), the magit/lazygit pattern
this follows: a toggle reveals hidden sections in place, no separate view.

## Scope
- **Keybind `a`** toggles archive visibility (session-only, in `?` help).
- **OFF (default)**: exactly today's view, plus the new hiding rule — a track
  whose subtree has zero open tasks but ≥1 archived one disappears from the
  list. A track with no tasks at all (never had any) still shows its dim
  "empty" row: intentional container vs archived-out noise.
- **ON**: each board grows a dim `Archived (N)` section at the bottom — rows
  show id, title, relative age; no gates/status columns (they're done by
  definition). Archived-out tracks reappear, dim, with their archived counts.
  `enter` on an archived row opens the normal read-only preview (glamour) —
  archived files render like any task file. No mutations ever offered on
  archived rows (they're read-only by convention — the CLI refuses their ids).
- **Cost discipline**: archive contents are read only when the toggle is ON
  (toggling on triggers the scan of archive/ dirs; OFF = zero extra reads —
  TestStartupPerformance untouched). The watcher already fires on tree
  changes; archived listings refresh with the normal re-scan while ON.
- Spec-of-record §8 + docs/tui.md updated (keys table, the hiding rule).
- Tests: toggle transitions, the two empty-track cases (never-had vs
  archived-out), archived section rendering + preview, zero reads while OFF
  (assert via the Mem fsys double if practical).

## Out of scope
Unarchiving from the TUI; CLI changes (get tasks still lists open only);
searching across archives; pagination (revisit if archives grow huge).

## Gates
- [x] Fixture with an archived-out track, a never-empty track, and mixed
  boards: OFF hides the archived-out track only; ON reveals sections +
  track, preview opens an archived task (frames in report).
- [x] Zero archive reads while OFF (test-proven); TestStartupPerformance
  green; frame audit green.
- [x] `just check` green; docs updated.

## Report

### 2026-07-18 02:23 — done

The archive is no longer invisible. `a` toggles it in place (magit-style, session-only, listed in `?`). Off, a track emptied by archiving drops out while a never-used one keeps its dim "empty" row. On, each board grows a dim "Archived (N)" section — id, title, age — archived-out tracks reappear with their counts, and enter opens the normal read-only preview. Archived file *contents* are read only while the toggle is on; off costs nothing (proven with a counting FS). Docs and tests updated.

### 2026-07-18 02:32

Applied backlog-archive simplify findings (commit 28e79fe).

Applied (behavior-preserving):
- tui/theme.go: dropped redundant StatusBacklog switch arms in statusInk + statusColor; the default t.Dim already yields the same value. Collapsed the stale comment.
- tui/entry.go: inlined single-caller archivedOut into visibleSubs (now '!showArchive && emptiedByArchiving'); emptiedByArchiving kept, still used by trackLine.

Skipped:
- trackInfo backlog counting (app/get.go): a behavior change to 'get tracks' output plus TrackInfo struct + test edits, not behavior-preserving. Latent bug worth a separate task.
- scanArchive -> tree.TaskFileNames and archivedID -> tree reuse: cross-layer refactors with divergent missing-dir semantics / new tree API; not clean behavior-preserving simplifications.
- taskLine/archivedLine and ArchivedCount/ArchivedSubtree dedup: confirmed not copy-paste / cemented by a test.

just check green, build ok.

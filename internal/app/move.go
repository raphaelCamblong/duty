package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Position names where Move places a task's row within its board once the
// track/section relocation is done: at the top of its section, or adjacent to
// a reference task's row (adopting that row's section). The zero Position asks
// for no reordering.
type Position struct {
	Top bool
	// Before is the id of the task whose row the moved row goes above; empty
	// when unused.
	Before string
	// After is the id of the task whose row the moved row goes below; empty
	// when unused.
	After string
}

func (p Position) None() bool { return !p.Top && p.Before == "" && p.After == "" }

// Move relocates a task. With a track path — relative to the tree root, "."
// naming the root board — the file is renamed into that track's folder (same
// filename — ids don't encode tracks), the source row is dropped and its
// section pruned, and a row is appended to the target's section (default
// "Open tasks") with the file's status preserved; all new contents are
// computed before the rename and the board writes. With an empty track the
// row moves under "## <section>" within its own board — the section is
// created above the footer when absent, and any section left empty is pruned.
// A non-zero pos then reorders the row within its board (a board-only edit).
// At least one of track, section, and pos must be non-empty.
func (a App) Move(cwd, id, track, section string, pos Position) error {
	if track == "" && section == "" && pos.None() {
		return errors.New("move: pass --track, --section, --top, --before, or --after")
	}
	root, taskPath, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	taskPath, err = a.relocate(root, id, taskPath, track, section)
	if err != nil {
		return err
	}
	if pos.None() {
		return nil
	}
	return a.reorderInBoard(root, taskPath, pos)
}

// relocate performs the track/section phase of a move and returns the task's
// resulting file path. With no track and no section it is a no-op returning
// taskPath unchanged — a position-only move.
func (a App) relocate(root, id, taskPath, track, section string) (string, error) {
	if track == "" && section == "" {
		return taskPath, nil
	}
	if track == "" {
		return taskPath, a.moveRowInBoard(taskPath, section)
	}
	if section == "" {
		section = board.DefaultSection
	}
	return a.moveTrack(root, id, taskPath, track, section)
}

// moveTrack relocates id's file into track's folder (dropping its source row,
// appending one to the target's section) and returns the file's new path.
func (a App) moveTrack(root, id, taskPath, track, section string) (string, error) {
	target, err := tree.ResolveTrack(a.fs, root, track)
	if err != nil {
		return "", err
	}
	if filepath.Dir(taskPath) == target {
		return taskPath, a.moveRowInBoard(taskPath, section)
	}
	t, _, err := a.readTask(taskPath)
	if err != nil {
		return "", err
	}
	filename := filepath.Base(taskPath)
	srcPath := boardBeside(taskPath)
	pruned, err := a.dropFromBoard(srcPath, filename)
	if err != nil {
		return "", err
	}
	if err := a.moveAcross(id, taskPath, target, section, pruned, t); err != nil {
		return "", err
	}
	return filepath.Join(target, filename), nil
}

func (a App) dropFromBoard(boardPath, filename string) ([]byte, error) {
	src, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return nil, err
	}
	dropped, err := board.DropRow(src, filename)
	if err != nil {
		return nil, err
	}
	return board.PruneEmptySections(dropped), nil
}

// moveAcross computes the target row before renaming the file, so a failure
// leaves the tree untouched.
func (a App) moveAcross(id, taskPath, target, section string, pruned []byte, t task.Task) error {
	filename := filepath.Base(taskPath)
	srcPath := boardBeside(taskPath)
	dstPath := boardIndexPath(target)
	dst, err := a.fs.ReadFile(dstPath)
	if err != nil {
		return err
	}
	withRow, err := board.AddRow(dst, section, t.ID, filename, t.Title, t.Status)
	if err != nil {
		return err
	}
	if err := a.fs.Rename(taskPath, filepath.Join(target, filename)); err != nil {
		return fmt.Errorf("move %s: %w", id, err)
	}
	if err := a.fs.WriteFile(srcPath, pruned); err != nil {
		return err
	}
	return a.fs.WriteFile(dstPath, withRow)
}

// moveRowInBoard moves the row of the task at taskPath under "## <section>"
// within its own board, byte-preserving the row line and pruning any section
// left empty. It is the same-board case of both --section and --track moves.
func (a App) moveRowInBoard(taskPath, section string) error {
	return a.applyEdit(boardBeside(taskPath), func(index []byte) ([]byte, error) {
		moved, err := board.MoveRow(index, filepath.Base(taskPath), section)
		if err != nil {
			return nil, err
		}
		return board.PruneEmptySections(moved), nil
	})
}

// reorderInBoard applies pos to the row of the task at taskPath — a board-only
// edit that relocates the row line, its bytes intact, leaving the task file
// untouched. --top lifts it to the top of its section; --before/--after place
// it adjacent to ref's row within the same board, adopting ref's section. A
// ref in another board is rejected, naming the fix: move --track first.
func (a App) reorderInBoard(root, taskPath string, pos Position) error {
	boardPath := boardBeside(taskPath)
	filename := filepath.Base(taskPath)
	return a.applyEdit(boardPath, func(index []byte) ([]byte, error) {
		return a.reorder(root, boardPath, index, filename, pos)
	})
}

func (a App) reorder(root, boardPath string, index []byte, filename string, pos Position) ([]byte, error) {
	if pos.Top {
		return board.ReorderTop(index, filename)
	}
	ref, adjacent := pos.Before, board.ReorderBefore
	if pos.After != "" {
		ref, adjacent = pos.After, board.ReorderAfter
	}
	refFile, err := a.refFilename(root, boardPath, ref)
	if err != nil {
		return nil, err
	}
	return adjacent(index, filename, refFile)
}

// refFilename resolves ref's filename, requiring it to live in the board at
// boardPath.
func (a App) refFilename(root, boardPath, ref string) (string, error) {
	refPath, err := tree.ResolveTask(a.fs, root, ref)
	if err != nil {
		return "", err
	}
	if boardBeside(refPath) != boardPath {
		return "", fmt.Errorf("%s is in another board — move --track it here first", ref)
	}
	return filepath.Base(refPath), nil
}

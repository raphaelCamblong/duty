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

// Dest is a move's track/section destination: a track path relative to the tree
// root ("." naming the root board, "" the task's current board) and the board
// section the row lands in ("" the default "Open tasks").
type Dest struct {
	Track   string
	Section string
}

func (d Dest) none() bool { return d.Track == "" && d.Section == "" }

// Move relocates id per dest (track/section) then reorders per pos, computing
// all writes before the rename; at least one of dest and pos must be set.
func (a App) Move(cwd, id string, dest Dest, pos Position) error {
	if dest.none() && pos.None() {
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
	taskPath, err = a.relocate(root, id, taskPath, dest)
	if err != nil {
		return err
	}
	if pos.None() {
		return nil
	}
	return a.reorderInBoard(root, taskPath, pos)
}

// relocate performs a move's track/section phase and returns the file's path,
// a no-op returning taskPath unchanged when dest is empty.
func (a App) relocate(root, id, taskPath string, dest Dest) (string, error) {
	if dest.none() {
		return taskPath, nil
	}
	if dest.Track == "" {
		return taskPath, a.moveRowInBoard(taskPath, dest.Section)
	}
	if dest.Section == "" {
		dest.Section = board.DefaultSection
	}
	return a.moveTrack(root, id, taskPath, dest)
}

// moveTrack relocates id's file into dest.Track's folder (dropping its source
// row, appending one to the target's section) and returns the file's new path.
func (a App) moveTrack(root, id, taskPath string, dest Dest) (string, error) {
	target, err := tree.ResolveTrack(a.fs, root, dest.Track)
	if err != nil {
		return "", err
	}
	if filepath.Dir(taskPath) == target {
		return taskPath, a.moveRowInBoard(taskPath, dest.Section)
	}
	parsed, _, err := a.readTask(taskPath)
	if err != nil {
		return "", err
	}
	filename := filepath.Base(taskPath)
	srcPath := boardBeside(taskPath)
	pruned, err := a.dropFromBoard(srcPath, filename)
	if err != nil {
		return "", err
	}
	if err := a.moveAcross(across{id: id, taskPath: taskPath, target: target, section: dest.Section, pruned: pruned, task: parsed}); err != nil {
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

// across is one cross-board move: the id and file of the task, the resolved
// target board dir and section its row lands in, the pruned source board bytes,
// and the parsed task whose row is written into the target.
type across struct {
	id       string
	taskPath string
	target   string
	section  string
	pruned   []byte
	task     task.Task
}

// moveAcross computes the target row before renaming the file, so a failure
// leaves the tree untouched.
func (a App) moveAcross(move across) error {
	filename := filepath.Base(move.taskPath)
	srcPath := boardBeside(move.taskPath)
	dstPath := boardIndexPath(move.target)
	dst, err := a.fs.ReadFile(dstPath)
	if err != nil {
		return err
	}
	withRow, err := board.AddRow(dst, move.section, board.Row{ID: move.task.ID, File: filename, Title: move.task.Title, Status: move.task.Status})
	if err != nil {
		return err
	}
	if err := a.fs.Rename(move.taskPath, filepath.Join(move.target, filename)); err != nil {
		return fmt.Errorf("move %s: %w", move.id, err)
	}
	if err := a.fs.WriteFile(srcPath, move.pruned); err != nil {
		return err
	}
	return a.fs.WriteFile(dstPath, withRow)
}

// moveRowInBoard moves the task's row under "## <section>" within its own board,
// byte-preserving the row and pruning any emptied section.
func (a App) moveRowInBoard(taskPath, section string) error {
	return a.applyEdit(boardBeside(taskPath), func(index []byte) ([]byte, error) {
		moved, err := board.MoveRow(index, filepath.Base(taskPath), section)
		if err != nil {
			return nil, err
		}
		return board.PruneEmptySections(moved), nil
	})
}

// reorderInBoard applies pos to the row at taskPath — a board-only edit, the
// task file untouched; a ref in another board is rejected (move --track first).
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

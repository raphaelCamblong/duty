package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Move relocates a task. With a track path — relative to the tree root, "."
// naming the root board — the file is renamed into that track's folder (same
// filename — ids don't encode tracks), the source row is dropped and its
// section pruned, and a row is appended to the target's section (default
// "Open tasks") with the file's status preserved; all new contents are
// computed before the rename and the board writes. With an empty track the
// row moves under "## <section>" within its own board — the section is
// created above the footer when absent, and any section left empty is
// pruned. At least one of track and section must be non-empty.
func (a App) Move(cwd, id, track, section string) error {
	if track == "" && section == "" {
		return errors.New("move: pass --track, --section, or both")
	}
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	if track == "" {
		return a.moveRow(cwd, id, section)
	}
	if section == "" {
		section = board.DefaultSection
	}
	return a.moveTrack(cwd, id, track, section)
}

// moveTrack relocates id's file into track's folder, dropping its source row
// and appending one to the target's section, the file's status preserved.
func (a App) moveTrack(cwd, id, track, section string) error {
	root, taskPath, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	target, err := tree.ResolveTrack(a.fs, root, track)
	if err != nil {
		return err
	}
	if filepath.Dir(taskPath) == target {
		return a.moveRowInBoard(taskPath, section)
	}
	t, _, err := a.readTask(taskPath)
	if err != nil {
		return err
	}
	filename := filepath.Base(taskPath)
	srcPath := boardBeside(taskPath)
	pruned, err := a.dropFromBoard(srcPath, filename)
	if err != nil {
		return err
	}
	return a.moveAcross(id, taskPath, target, section, pruned, t)
}

// dropFromBoard returns the board index at boardPath with filename's row
// dropped and any section left empty pruned.
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

// moveAcross renames the task file into target and writes both boards; the
// target row is computed before the rename so a failure leaves the tree
// untouched.
func (a App) moveAcross(id, taskPath, target, section string, pruned []byte, t task.Task) error {
	filename := filepath.Base(taskPath)
	srcPath := boardBeside(taskPath)
	dstPath := filepath.Join(target, names.BoardFile)
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

// moveRow moves a task's board row under "## <section>" line-surgically.
// Sections live only on the board — the task file carries none — so this is
// the one mutation that touches a single file.
func (a App) moveRow(cwd, id, section string) error {
	taskPath, err := a.resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	return a.moveRowInBoard(taskPath, section)
}

// moveRowInBoard moves the row of the task at taskPath under "## <section>"
// within its own board, byte-preserving the row line and pruning any section
// left empty. It is the same-board case of both --section and --track moves.
func (a App) moveRowInBoard(taskPath, section string) error {
	boardPath := boardBeside(taskPath)
	index, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return err
	}
	moved, err := board.MoveRow(index, filepath.Base(taskPath), section)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(boardPath, board.PruneEmptySections(moved))
}

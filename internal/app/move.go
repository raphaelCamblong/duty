package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

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
	if track == "" {
		return a.moveRow(cwd, id, section)
	}
	if section == "" {
		section = board.DefaultSection
	}

	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return err
	}
	taskPath, err := tree.ResolveTask(a.fs, root, id)
	if err != nil {
		return err
	}
	target, err := a.targetBoard(root, track)
	if err != nil {
		return err
	}

	content, err := a.fs.ReadFile(taskPath)
	if err != nil {
		return err
	}
	t, err := task.Parse(content)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	filename := filepath.Base(taskPath)
	srcDir := filepath.Dir(taskPath)
	srcPath := filepath.Join(srcDir, names.BoardFile)
	src, err := a.fs.ReadFile(srcPath)
	if err != nil {
		return err
	}
	dropped, err := board.DropRow(src, filename)
	if err != nil {
		return err
	}
	pruned := board.PruneEmptySections(dropped)

	if srcDir == target {
		withRow, err := board.AddRow(pruned, section, t.ID, filename, t.Title, t.Status)
		if err != nil {
			return err
		}
		return a.fs.WriteFile(srcPath, withRow)
	}

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
	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
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

// targetBoard resolves track — relative to root, "." meaning the root
// board — to an existing board directory: one holding a board index inside
// the tree.
func (a App) targetBoard(root, track string) (string, error) {
	if filepath.IsAbs(track) {
		return "", fmt.Errorf("track path %q must be relative to the tree root", track)
	}
	dir := filepath.Join(root, filepath.FromSlash(track))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("track path %q escapes the tree", track)
	}
	info, err := a.fs.Stat(filepath.Join(dir, names.BoardFile))
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("no track at %q: no %s there", track, names.BoardFile)
	}
	return dir, nil
}

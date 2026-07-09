package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// Move moves a task to another board: the file is renamed into the target
// board's folder (same filename — ids don't encode boards), the source row is
// dropped and its section pruned, and a row is appended to the target's
// section with the file's status preserved. boardPath is relative to the
// tree root; "." names the root board; an empty section means the default
// one. All new contents are computed before the rename and the board writes.
func (a App) Move(cwd, id, boardPath, section string) error {
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
	target, err := a.targetBoard(root, boardPath)
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

// targetBoard resolves boardPath — relative to root, "." meaning the root
// board — to an existing board directory: one holding a board index inside
// the tree.
func (a App) targetBoard(root, boardPath string) (string, error) {
	if filepath.IsAbs(boardPath) {
		return "", fmt.Errorf("board path %q must be relative to the tree root", boardPath)
	}
	dir := filepath.Join(root, filepath.FromSlash(boardPath))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("board path %q escapes the tree", boardPath)
	}
	info, err := a.fs.Stat(filepath.Join(dir, names.BoardFile))
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("no board at %q: no %s there", boardPath, names.BoardFile)
	}
	return dir, nil
}

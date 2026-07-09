package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsutil"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

const moveUsage = "usage: duty move <id> <board-path> [--section NAME]"

// runMove moves a task to another board: the file is renamed into the target
// board's folder (same filename — ids don't encode boards), the source row is
// dropped and its section pruned, and a row is appended to the target's
// section with the file's status preserved. board-path is relative to the
// tree root; "." names the root board. All new contents are computed before
// the rename and the board writes.
func runMove(cwd string, args []string) error {
	fs := flag.NewFlagSet("move", flag.ContinueOnError)
	section := fs.String("section", board.DefaultSection, "target board section for the row")
	pos, err := positionals(fs, args, moveUsage)
	if err != nil {
		return err
	}
	if len(pos) != 2 || pos[0] == "" || pos[1] == "" {
		return errors.New(moveUsage)
	}
	id, boardPath := pos[0], pos[1]
	sect := *section
	if sect == "" {
		sect = board.DefaultSection
	}

	root, err := tree.FindRoot(cwd)
	if err != nil {
		return err
	}
	taskPath, err := tree.ResolveTask(root, id)
	if err != nil {
		return err
	}
	target, err := targetBoard(root, boardPath)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(taskPath)
	if err != nil {
		return err
	}
	t, err := task.Parse(content)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	filename := filepath.Base(taskPath)
	srcDir := filepath.Dir(taskPath)
	srcPath := filepath.Join(srcDir, boardFile)
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	dropped, err := board.DropRow(src, filename)
	if err != nil {
		return err
	}
	pruned := board.PruneEmptySections(dropped)

	if srcDir == target {
		withRow, err := board.AddRow(pruned, sect, t.ID, filename, t.Title, t.Status)
		if err != nil {
			return err
		}
		return fsutil.WriteAtomic(srcPath, withRow)
	}

	dstPath := filepath.Join(target, boardFile)
	dst, err := os.ReadFile(dstPath)
	if err != nil {
		return err
	}
	withRow, err := board.AddRow(dst, sect, t.ID, filename, t.Title, t.Status)
	if err != nil {
		return err
	}
	if err := os.Rename(taskPath, filepath.Join(target, filename)); err != nil {
		return fmt.Errorf("move %s: %w", id, err)
	}
	if err := fsutil.WriteAtomic(srcPath, pruned); err != nil {
		return err
	}
	return fsutil.WriteAtomic(dstPath, withRow)
}

// targetBoard resolves boardPath — relative to root, "." meaning the root
// board — to an existing board directory: one holding a BOARD.md inside the
// tree.
func targetBoard(root, boardPath string) (string, error) {
	if filepath.IsAbs(boardPath) {
		return "", fmt.Errorf("board path %q must be relative to the tree root", boardPath)
	}
	dir := filepath.Join(root, filepath.FromSlash(boardPath))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("board path %q escapes the tree", boardPath)
	}
	info, err := os.Stat(filepath.Join(dir, boardFile))
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("no board at %q: no %s there", boardPath, boardFile)
	}
	return dir, nil
}

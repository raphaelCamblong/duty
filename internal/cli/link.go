package cli

import (
	"errors"
	"flag"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
)

const linkUsage = "usage: duty link <id> <section>"

// runLink moves a task's board row under "## <section>", creating the section
// above the footer when absent and pruning any section left empty. Sections
// live only on the board — the task file carries none — so this is the one
// mutation that touches a single file.
func runLink(f fsys.FS, cwd string, args []string) error {
	set := flag.NewFlagSet("link", flag.ContinueOnError)
	pos, err := positionals(set, args, linkUsage)
	if err != nil {
		return err
	}
	if len(pos) != 2 || pos[0] == "" || pos[1] == "" {
		return errors.New(linkUsage)
	}
	id, section := pos[0], pos[1]

	taskPath, err := resolveOpen(f, cwd, id)
	if err != nil {
		return err
	}
	boardPath := filepath.Join(filepath.Dir(taskPath), names.BoardFile)
	index, err := f.ReadFile(boardPath)
	if err != nil {
		return err
	}
	moved, err := board.MoveRow(index, filepath.Base(taskPath), section)
	if err != nil {
		return err
	}
	return f.WriteFile(boardPath, board.PruneEmptySections(moved))
}

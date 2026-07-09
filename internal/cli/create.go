package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

const createUsage = "usage: duty create <title> [--slug S] [--blocked-by ID]... [--section NAME]"

// runCreate creates a task in the current board: it validates every
// --blocked-by id against the whole tree, numbers the task tree-wide, writes
// the template file and appends the board row (status todo) in one command,
// then prints the created path — the only output.
func runCreate(f fsys.FS, cwd string, args []string, stdout io.Writer) error {
	set := flag.NewFlagSet("create", flag.ContinueOnError)
	slug := set.String("slug", "", "filename slug override")
	section := set.String("section", board.DefaultSection, "board section for the new row")
	var blockedBy stringList
	set.Var(&blockedBy, "blocked-by", "id of a task that must be done first (repeatable)")
	pos, err := positionals(set, args, createUsage)
	if err != nil {
		return err
	}
	if len(pos) != 1 || pos[0] == "" {
		return errors.New(createUsage)
	}
	title := pos[0]
	if *slug != "" && !nameRE.MatchString(*slug) {
		return fmt.Errorf("invalid slug %q: must match [a-z0-9-]+", *slug)
	}

	boardDir, err := tree.CurrentBoard(f, cwd)
	if err != nil {
		return err
	}
	root, err := tree.FindRoot(f, cwd)
	if err != nil {
		return err
	}
	for _, dep := range blockedBy {
		if _, err := tree.ResolveTask(f, root, dep); err != nil && !errors.Is(err, tree.ErrArchived) {
			return fmt.Errorf("blocked-by: %w", err)
		}
	}
	nn, err := tree.NextNN(f, root)
	if err != nil {
		return err
	}
	id := "T-" + nn
	s := *slug
	if s == "" {
		s = task.Slugify(title)
	}
	if s == "" {
		return fmt.Errorf("cannot derive a slug from %q, pass --slug", title)
	}
	sect := *section
	if sect == "" {
		sect = board.DefaultSection
	}

	filename := id + "-" + s + ".md"
	boardPath := filepath.Join(boardDir, names.BoardFile)
	content, err := f.ReadFile(boardPath)
	if err != nil {
		return err
	}
	withRow, err := board.AddRow(content, sect, id, filename, title, task.StatusTodo)
	if err != nil {
		return err
	}
	taskPath := filepath.Join(boardDir, filename)
	if err := f.WriteFile(taskPath, task.Render(id, title, blockedBy)); err != nil {
		return err
	}
	if err := f.WriteFile(boardPath, withRow); err != nil {
		return err
	}
	fmt.Fprintln(stdout, taskPath)
	return nil
}

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

// CreateTask creates a task in the board containing cwd and returns the new
// file's path: every blocked-by id is validated against the whole tree, the
// task is numbered tree-wide, and the template file and its board row
// (status todo) are written in one use-case. An empty slug is derived from
// the title; an empty section means the default one.
func (a App) CreateTask(cwd, title, slug, section string, blockedBy []string) (string, error) {
	if slug != "" && !task.ValidSlug(slug) {
		return "", fmt.Errorf("invalid slug %q: want 1-40 chars of [a-z0-9-], no leading or trailing hyphen", slug)
	}
	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return "", err
	}
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", err
	}
	if err := a.validateBlockedBy(root, blockedBy); err != nil {
		return "", err
	}
	nn, err := tree.NextNN(a.fs, root)
	if err != nil {
		return "", err
	}
	if slug == "" {
		slug = task.Slugify(title)
	}
	if slug == "" {
		return "", fmt.Errorf("cannot derive a slug from %q, pass --slug", title)
	}
	if section == "" {
		section = board.DefaultSection
	}
	return a.writeTask(boardDir, task.FormatID(nn), slug, title, section, blockedBy)
}

// validateBlockedBy checks every dependency id resolves somewhere in the
// tree; archived dependencies are legal.
func (a App) validateBlockedBy(root string, blockedBy []string) error {
	for _, dep := range blockedBy {
		if _, err := tree.ResolveTask(a.fs, root, dep); err != nil && !errors.Is(err, tree.ErrArchived) {
			return fmt.Errorf("blocked-by: %w", err)
		}
	}
	return nil
}

// writeTask renders the template file and appends its board row (status
// todo), both contents computed before either write, and returns the new
// file's path.
func (a App) writeTask(boardDir, id, slug, title, section string, blockedBy []string) (string, error) {
	filename := id + "-" + slug + ".md"
	boardPath := filepath.Join(boardDir, names.BoardFile)
	content, err := a.fs.ReadFile(boardPath)
	if err != nil {
		return "", err
	}
	withRow, err := board.AddRow(content, section, id, filename, title, task.StatusTodo)
	if err != nil {
		return "", err
	}
	taskPath := filepath.Join(boardDir, filename)
	if err := a.fs.WriteFile(taskPath, task.Render(id, title, blockedBy)); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(boardPath, withRow); err != nil {
		return "", err
	}
	return taskPath, nil
}

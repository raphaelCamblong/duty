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
	if slug != "" && !nameRE.MatchString(slug) {
		return "", fmt.Errorf("invalid slug %q: must match [a-z0-9-]+", slug)
	}

	boardDir, err := tree.CurrentBoard(a.fs, cwd)
	if err != nil {
		return "", err
	}
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", err
	}
	for _, dep := range blockedBy {
		if _, err := tree.ResolveTask(a.fs, root, dep); err != nil && !errors.Is(err, tree.ErrArchived) {
			return "", fmt.Errorf("blocked-by: %w", err)
		}
	}
	nn, err := tree.NextNN(a.fs, root)
	if err != nil {
		return "", err
	}
	id := "T-" + nn
	if slug == "" {
		slug = task.Slugify(title)
	}
	if slug == "" {
		return "", fmt.Errorf("cannot derive a slug from %q, pass --slug", title)
	}
	if section == "" {
		section = board.DefaultSection
	}

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

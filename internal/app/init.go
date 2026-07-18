package app

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/names"
	"github.com/raphaelCamblong/duty/internal/tree"
)

//go:embed readme.md.tmpl
var readmeTmplText string

// readmeTmpl renders the agent-facing convention page (spec §9): the command
// table, the lifecycle→command mapping, and what stays the worker's judgment.
var readmeTmpl = template.Must(template.New("readme").Parse(readmeTmplText))

// renderReadme produces the convention readme Init drops next to the root board.
func renderReadme() []byte {
	var b bytes.Buffer
	_ = readmeTmpl.Execute(&b, struct{ Board string }{Board: names.BoardFile})
	return b.Bytes()
}

// Init bootstraps a duty tree in cwd: duty/ with a skeleton board index
// (H1 = title, default "Board"), the convention readme, and archive/. It
// refuses to run inside an existing tree, and on success returns the tree
// directory's path.
func (a App) Init(cwd, title string) (string, error) {
	if title == "" {
		title = "Board"
	}
	if dir, err := tree.CurrentBoard(a.fs, cwd); err == nil {
		return "", fmt.Errorf("already inside a duty tree (%s)", dir)
	}
	dir := filepath.Join(cwd, names.TreeDir)
	if err := a.fs.MkdirAll(filepath.Join(dir, names.ArchiveDir)); err != nil {
		return "", fmt.Errorf("init: %w", err)
	}
	if err := a.fs.WriteFile(boardIndexPath(dir), board.Render(title)); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(filepath.Join(dir, names.ReadmeFile), renderReadme()); err != nil {
		return "", err
	}
	return dir, nil
}

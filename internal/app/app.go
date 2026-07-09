// Package app implements duty's use-cases: one method per verb, all
// orchestration over an injected fsys.FS. The sync invariant lives here —
// every mutating use-case edits the task file AND its board row in one call.
// Methods return data and errors; they never print and never parse flags.
package app

import (
	"fmt"
	"regexp"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tree"
)

// App bundles duty's use-cases over one filesystem.
type App struct {
	fs fsys.FS
}

// New returns an App operating on f.
func New(f fsys.FS) App {
	return App{fs: f}
}

// nameRE validates sub-board folder names and task filename slugs.
var nameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// unknownStatusErr is the one-line error every use-case rejecting an unknown
// status string returns.
func unknownStatusErr(status string) error {
	return fmt.Errorf("unknown status %q: want todo, in-progress, done or blocked", status)
}

// resolveOpen resolves id to its open task file anywhere in the tree
// containing cwd. Archived ids fail with tree.ErrArchived in the chain:
// archived tasks are read-only.
func (a App) resolveOpen(cwd, id string) (string, error) {
	root, err := tree.FindRoot(a.fs, cwd)
	if err != nil {
		return "", err
	}
	return tree.ResolveTask(a.fs, root, id)
}

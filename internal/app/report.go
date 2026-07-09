package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Report appends the text read from r under the task's "## Report" heading,
// creating the heading once at the end of the file. Reports accumulate —
// nothing already in the file is rewritten. Empty (or blank) input is
// refused; r is read only after the id resolves.
func (a App) Report(cwd, id string, r io.Reader) error {
	taskPath, err := a.resolveOpen(cwd, id)
	if err != nil {
		return err
	}
	text, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(bytes.TrimSpace(text)) == 0 {
		return errors.New("empty report: pipe the report text on stdin")
	}
	content, err := a.fs.ReadFile(taskPath)
	if err != nil {
		return err
	}
	return a.fs.WriteFile(taskPath, task.AppendReport(content, text))
}

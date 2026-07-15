package app

import (
	"bytes"
	"fmt"
	"io"

	"github.com/raphaelCamblong/duty/internal/task"
)

// Report appends the text read from r under the task's "## Report" heading,
// opened by a dated "### 2006-01-02 15:04" line (plus " — status" when status
// is given) stamped from a.now, creating the heading once at the end of the
// file. Reports accumulate — nothing already in the file is rewritten. When
// status is non-empty it also flips the task's status (file + board row) in
// the same locked write, the done/blocked lifecycle endings: both edits are
// computed before either file is written, so any error leaves neither the
// report nor the status applied. Empty (or blank) input is refused; r is
// read only after the id resolves.
func (a App) Report(cwd, id string, r io.Reader, status string, force bool) error {
	if status != "" && !task.ValidStatus(status) {
		return unknownStatusErr(status)
	}
	root, taskPath, err := a.resolveOpenWithRoot(cwd, id)
	if err != nil {
		return err
	}
	text, err := readNonBlank(r, "report")
	if err != nil {
		return err
	}
	unlock, err := a.lock(root)
	if err != nil {
		return err
	}
	defer unlock()
	content, err := a.fs.ReadFile(taskPath)
	if err != nil {
		return err
	}
	withReport := task.AppendReport(content, task.ReportBlock(a.now(), status, text))
	if status == "" {
		return a.fs.WriteFile(taskPath, withReport)
	}
	t, err := task.Parse(content)
	if err != nil {
		return fmt.Errorf("%s: %w", taskPath, err)
	}
	return a.statusWrite(taskPath, id, status, force, withReport, t.Status)
}

// readNonBlank reads all of r and rejects blank input, naming the piped content
// kind in both the read-error and empty-input messages.
func readNonBlank(r io.Reader, kind string) ([]byte, error) {
	text, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(bytes.TrimSpace(text)) == 0 {
		return nil, fmt.Errorf("empty %s: pipe the %s text on stdin", kind, kind)
	}
	return text, nil
}

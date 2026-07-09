package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/raphaelCamblong/duty/internal/fsutil"
	"github.com/raphaelCamblong/duty/internal/task"
)

const reportUsage = "usage: duty report <id>"

// runReport appends stdin under the task's "## Report" heading, creating the
// heading once at the end of the file. Reports accumulate — nothing already
// in the file is rewritten. Empty (or blank) stdin is refused.
func runReport(cwd string, args []string, stdin io.Reader) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	pos, err := positionals(fs, args, reportUsage)
	if err != nil {
		return err
	}
	if len(pos) != 1 || pos[0] == "" {
		return errors.New(reportUsage)
	}

	taskPath, err := resolveOpen(cwd, pos[0])
	if err != nil {
		return err
	}
	text, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(bytes.TrimSpace(text)) == 0 {
		return errors.New("empty report: pipe the report text on stdin")
	}
	content, err := os.ReadFile(taskPath)
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(taskPath, task.AppendReport(content, text))
}

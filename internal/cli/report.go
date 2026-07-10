package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	reportUsage   = "usage: duty report <id>"
	reportExample = `  duty report T-07 < report.txt`
)

// newReportCmd builds the report command: append stdin under the task's
// "## Report" heading.
func newReportCmd(a app.App, cwd string, stdin io.Reader) *cobra.Command {
	return &cobra.Command{
		Use:     "report <id>",
		Short:   "append stdin to a task's report",
		Example: reportExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(reportUsage)
			}
			return a.Report(cwd, args[0], stdin)
		},
	}
}

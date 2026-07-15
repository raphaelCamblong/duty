package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	reportUsage   = "usage: duty report <id> [--status S]"
	reportExample = "  duty report T-07 --status done < report.txt\n  duty report T-07 < report.txt"
)

// newReportCmd builds the report command: append stdin under the task's
// "## Report" heading, optionally flipping the status in the same write.
func newReportCmd(a app.App, cwd string, stdin io.Reader) *cobra.Command {
	var (
		status string
		force  bool
	)
	cmd := &cobra.Command{
		Use:     "report <id>",
		Short:   "append stdin to a task's report, optionally setting its status",
		Example: reportExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(reportUsage)
			}
			return a.Report(cwd, args[0], stdin, status, force)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "also set the task's status (file + board) in one write")
	cmd.Flags().BoolVar(&force, "force", false, "with --status in-progress, take over an existing claim")
	return cmd
}

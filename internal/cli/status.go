package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const statusUsage = "usage: duty status <id> <status>"

// newStatusCmd builds the status command: set a task's status in its file
// and board row.
func newStatusCmd(a app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:   "status <id> <status>",
		Short: "set a task's status in its file and board row",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(statusUsage)
			}
			return a.SetStatus(cwd, args[0], args[1])
		},
	}
}

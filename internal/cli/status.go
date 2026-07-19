package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	statusUsage   = "usage: duty status <id> <status>"
	statusExample = `  duty status T-07 in-progress`
)

func newStatusCmd(a app.App, cwd string) *cobra.Command {
	var (
		force bool
		as    string
	)
	cmd := &cobra.Command{
		Use:     "status <id> <status>",
		Short:   "set a task's status in its file and board row",
		Example: statusExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(statusUsage)
			}
			return a.SetStatus(cwd, args[0], args[1], force, claimer(as))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "take over a task already in-progress")
	addAsFlag(cmd, &as)
	return cmd
}

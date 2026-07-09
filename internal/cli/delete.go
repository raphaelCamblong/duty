package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const deleteUsage = "usage: duty delete <id> [--force]"

// newDeleteCmd builds the delete command: remove an open task and its row.
func newDeleteCmd(a app.App, cwd string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "remove an open task and its board row",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(deleteUsage)
			}
			return a.Delete(cwd, args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "allow deleting a done task")
	return cmd
}

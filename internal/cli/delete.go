package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const deleteTaskUsage = "usage: duty delete task <id> [--force]"

// newDeleteCmd builds the delete verb: resource subcommands for removal.
func newDeleteCmd(a app.App, cwd string) *cobra.Command {
	cmd := newGroupCmd("delete", "remove a task", deleteTaskUsage)
	cmd.AddCommand(newDeleteTaskCmd(a, cwd))
	return cmd
}

// newDeleteTaskCmd builds delete task: remove an open task and its row.
func newDeleteTaskCmd(a app.App, cwd string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "task <id>",
		Short: "remove an open task and its board row",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(deleteTaskUsage)
			}
			return a.Delete(cwd, args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "allow deleting a done task")
	return cmd
}

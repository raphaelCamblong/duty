package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const moveUsage = "usage: duty move <id> <board-path> [--section NAME]"

// newMoveCmd builds the move command: move a task to another board.
func newMoveCmd(a app.App, cwd string) *cobra.Command {
	var section string
	cmd := &cobra.Command{
		Use:   "move <id> <board-path>",
		Short: "move a task to another board",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(moveUsage)
			}
			return a.Move(cwd, args[0], args[1], section)
		},
	}
	cmd.Flags().StringVar(&section, "section", "", `target board section for the row (default "Open tasks")`)
	return cmd
}

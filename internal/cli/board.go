package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const boardUsage = "usage: duty board <name> [--title T]"

// newBoardCmd builds the board command: new sub-board under the current one.
func newBoardCmd(a app.App, cwd string) *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "board <name>",
		Short: "create a sub-board under the current board",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New(boardUsage)
			}
			return a.CreateBoard(cwd, args[0], title)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "board title (default: the name)")
	return cmd
}

package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const linkUsage = "usage: duty link <id> <section>"

// newLinkCmd builds the link command: move a task's board row to a section.
func newLinkCmd(a app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:   "link <id> <section>",
		Short: "move a task's board row under a section",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(linkUsage)
			}
			return a.Link(cwd, args[0], args[1])
		},
	}
}

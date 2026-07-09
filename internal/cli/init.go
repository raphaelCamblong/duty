package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const initUsage = "usage: duty init [title]"

// newInitCmd builds the init command: bootstrap a duty tree in cwd.
func newInitCmd(a app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:   "init [title]",
		Short: "bootstrap a duty tree in the current directory",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New(initUsage)
			}
			title := ""
			if len(args) == 1 {
				title = args[0]
			}
			return a.Init(cwd, title)
		},
	}
}

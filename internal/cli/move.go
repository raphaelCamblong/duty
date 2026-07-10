package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	moveUsage   = "usage: duty move <id> [--track PATH] [--section NAME] (at least one flag)"
	moveExample = `  duty move T-07 --track backend --section "Open tasks"`
)

// newMoveCmd builds the move command: relocate a task to another track,
// move its board row to a section, or both.
func newMoveCmd(a app.App, cwd string) *cobra.Command {
	var (
		track   string
		section string
	)
	cmd := &cobra.Command{
		Use:     "move <id>",
		Short:   "move a task to another track and/or board section",
		Example: moveExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" || (track == "" && section == "") {
				return errors.New(moveUsage)
			}
			return a.Move(cwd, args[0], track, section)
		},
	}
	cmd.Flags().StringVar(&track, "track", "", `target track path from the tree root ("." = root board)`)
	cmd.Flags().StringVar(&section, "section", "", "target board section for the row")
	return cmd
}

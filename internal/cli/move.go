package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	moveUsage   = "usage: duty move <id> [--track PATH] [--section NAME] [--top | --before REF | --after REF] (at least one flag)"
	moveExample = `  duty move T-07 --track backend --section "Open tasks"
  duty move T-07 --top
  duty move T-07 --after T-03`
)

func newMoveCmd(svc app.App, cwd string) *cobra.Command {
	var (
		track   string
		section string
		pos     app.Position
	)
	cmd := &cobra.Command{
		Use:     "move <id>",
		Short:   "move a task across tracks and sections, or reorder its row",
		Example: moveExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" || (track == "" && section == "" && pos.None()) {
				return errors.New(moveUsage)
			}
			return svc.Move(cwd, args[0], app.Dest{Track: track, Section: section}, pos)
		},
	}
	cmd.Flags().StringVar(&track, "track", "", `target track path from the tree root ("." = root board)`)
	cmd.Flags().StringVar(&section, "section", "", "target board section for the row")
	cmd.Flags().BoolVar(&pos.Top, "top", false, "move the row to the top of its section")
	cmd.Flags().StringVar(&pos.Before, "before", "", "move the row just above REF's row (adopting REF's section)")
	cmd.Flags().StringVar(&pos.After, "after", "", "move the row just below REF's row (adopting REF's section)")
	cmd.MarkFlagsMutuallyExclusive("top", "before", "after")
	return cmd
}

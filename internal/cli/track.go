package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const trackUsage = "usage: duty track <name> [--title T]"

// newTrackCmd builds the track command: new track (a folder with its own
// board) under the current one. "board" stays as a working alias.
func newTrackCmd(a app.App, cwd string) *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:     "track <name>",
		Aliases: []string{"board"},
		Short:   "create a track under the current one",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New(trackUsage)
			}
			return a.CreateBoard(cwd, args[0], title)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "track title (default: the name)")
	return cmd
}

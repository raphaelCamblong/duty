package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tui"
)

const (
	tuiUsage   = "usage: duty tui"
	tuiExample = `  duty tui`
)

func newTUICmd(fs fsys.FS, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:     "tui",
		Short:   "launch the live board viewer",
		Example: tuiExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(tuiUsage)
			}
			return tui.Run(fs, cwd)
		},
	}
}

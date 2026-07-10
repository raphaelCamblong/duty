package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	archiveUsage   = "usage: duty archive"
	archiveExample = `  duty archive`
)

// newArchiveCmd builds the archive command: move every done task in the
// current board and below into its own board's archive/.
func newArchiveCmd(a app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:     "archive",
		Short:   "archive every done task in the current board and below",
		Example: archiveExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(archiveUsage)
			}
			return a.Archive(cwd)
		},
	}
}

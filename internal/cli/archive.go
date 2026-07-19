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

func newArchiveCmd(svc app.App, cwd string) *cobra.Command {
	var in string
	cmd := &cobra.Command{
		Use:     "archive",
		Short:   "archive every done task in the current board and below",
		Example: archiveExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(archiveUsage)
			}
			return svc.Archive(app.Scope{Cwd: cwd, In: in})
		},
	}
	addInFlag(cmd, &in)
	return cmd
}

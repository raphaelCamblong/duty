package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	initUsage   = "usage: duty init [title]"
	initExample = `  duty init "Q3 roadmap"`
)

// newInitCmd builds the init command: bootstrap a duty tree in cwd, printing
// where it landed.
func newInitCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "init [title]",
		Short:   "bootstrap a duty tree in the current directory",
		Example: initExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New(initUsage)
			}
			title := ""
			if len(args) == 1 {
				title = args[0]
			}
			path, err := a.Init(cwd, title)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "initialized duty tree in %s\n", path)
			return nil
		},
	}
}

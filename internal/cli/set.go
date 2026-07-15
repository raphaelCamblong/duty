package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	setUsage   = "usage: duty set <id> <section>"
	setExample = `  duty set T-07 goal < goal.md`
)

// newSetCmd builds the set command: replace a task section's body from stdin.
func newSetCmd(a app.App, cwd string, stdin io.Reader) *cobra.Command {
	return &cobra.Command{
		Use:     "set <id> <section>",
		Short:   "replace a task section's body from stdin",
		Example: setExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(setUsage)
			}
			return a.SetSection(cwd, args[0], args[1], stdin)
		},
	}
}

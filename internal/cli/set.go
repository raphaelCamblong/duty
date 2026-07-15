package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	setUsage   = "usage: duty set <id> [section]"
	setExample = "  duty set T-07 goal < goal.md\n  duty set T-07 < body.md"
)

// newSetCmd builds the set command: replace one section's body (with a section
// argument) or every "## " block on stdin (without one) from stdin.
func newSetCmd(a app.App, cwd string, stdin io.Reader) *cobra.Command {
	return &cobra.Command{
		Use:     "set <id> [section]",
		Short:   "replace one or more task sections from stdin",
		Example: setExample,
		RunE: func(_ *cobra.Command, args []string) error {
			switch {
			case len(args) == 2 && args[0] != "" && args[1] != "":
				return a.SetSection(cwd, args[0], args[1], stdin)
			case len(args) == 1 && args[0] != "":
				return a.SetSections(cwd, args[0], stdin)
			default:
				return errors.New(setUsage)
			}
		},
	}
}

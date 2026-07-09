package cli

import (
	"errors"
	"flag"

	"github.com/raphaelCamblong/duty/internal/tui"
)

const tuiUsage = "usage: duty tui"

// runTUI launches the live board viewer on the tree containing cwd.
func runTUI(cwd string, args []string) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	pos, err := positionals(fs, args, tuiUsage)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return errors.New(tuiUsage)
	}
	return tui.Run(cwd)
}

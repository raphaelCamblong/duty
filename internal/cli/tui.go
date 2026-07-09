package cli

import (
	"errors"
	"flag"

	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tui"
)

const tuiUsage = "usage: duty tui"

// runTUI launches the live board viewer on the tree containing cwd.
func runTUI(f fsys.FS, cwd string, args []string) error {
	set := flag.NewFlagSet("tui", flag.ContinueOnError)
	pos, err := positionals(set, args, tuiUsage)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return errors.New(tuiUsage)
	}
	return tui.Run(f, cwd)
}

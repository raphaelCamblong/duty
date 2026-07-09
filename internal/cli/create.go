package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const createUsage = "usage: duty create <title> [--slug S] [--blocked-by ID]... [--section NAME]"

// newCreateCmd builds the create command: new task in the current board,
// printing the created path — the only output.
func newCreateCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var (
		slug      string
		section   string
		blockedBy stringList
	)
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "create a task in the current board",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(createUsage)
			}
			path, err := a.CreateTask(cwd, args[0], slug, section, blockedBy)
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "filename slug override")
	cmd.Flags().StringVar(&section, "section", "", `board section for the new row (default "Open tasks")`)
	cmd.Flags().Var(&blockedBy, "blocked-by", "id of a task that must be done first (repeatable)")
	return cmd
}

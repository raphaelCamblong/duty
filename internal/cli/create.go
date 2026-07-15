package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	createUsage        = "usage: duty create <task|track> [args]"
	createExample      = "  duty create task \"Fix login\" --blocked-by T-03\n  duty create track backend --title \"Backend work\""
	createTaskUsage    = "usage: duty create task <title> [--slug S] [--blocked-by ID]... [--section NAME] [--body]"
	createTaskExample  = `  duty create task "Fix login" --blocked-by T-03`
	createTrackUsage   = "usage: duty create track <name> [--title T]"
	createTrackExample = `  duty create track backend --title "Backend work"`
)

// newCreateCmd builds the create verb: resource subcommands for tasks and
// tracks.
func newCreateCmd(a app.App, cwd string, stdin io.Reader, stdout io.Writer) *cobra.Command {
	cmd := newGroupCmd("create", "create a task or a track", createUsage, createExample)
	cmd.AddCommand(
		newCreateTaskCmd(a, cwd, stdin, stdout),
		newCreateTrackCmd(a, cwd),
	)
	return cmd
}

// newCreateTaskCmd builds create task: new task in the current board,
// printing the created path — the only output. With --body the whole markdown
// body (## sections) is read from stdin instead of the section skeleton.
func newCreateTaskCmd(a app.App, cwd string, stdin io.Reader, stdout io.Writer) *cobra.Command {
	var (
		slug      string
		section   string
		in        string
		body      bool
		blockedBy stringList
	)
	cmd := &cobra.Command{
		Use:     "task <title>",
		Short:   "create a task in the current board",
		Example: createTaskExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(createTaskUsage)
			}
			var bodyReader io.Reader
			if body {
				bodyReader = stdin
			}
			path, err := a.CreateTask(cwd, args[0], slug, section, in, blockedBy, bodyReader)
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "filename slug override")
	cmd.Flags().StringVar(&section, "section", "", `board section for the new row (default "Open tasks")`)
	cmd.Flags().BoolVar(&body, "body", false, "read the task body (## sections) from stdin, below a generated H1")
	cmd.Flags().Var(&blockedBy, "blocked-by", "id of a task that must be done first (repeatable)")
	addInFlag(cmd, &in)
	return cmd
}

// newCreateTrackCmd builds create track: new track (a folder with its own
// board) under the current one.
func newCreateTrackCmd(a app.App, cwd string) *cobra.Command {
	var (
		title string
		in    string
	)
	cmd := &cobra.Command{
		Use:     "track <name>",
		Short:   "create a track under the current board",
		Example: createTrackExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New(createTrackUsage)
			}
			return a.CreateTrack(cwd, args[0], title, in)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "track title (default: the name)")
	addInFlag(cmd, &in)
	return cmd
}

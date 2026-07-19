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

func newCreateCmd(svc app.App, cwd string, stdin io.Reader, stdout io.Writer) *cobra.Command {
	cmd := newGroupCmd("create", "create a task or a track", createUsage, createExample)
	cmd.AddCommand(
		newCreateTaskCmd(svc, cwd, stdin, stdout),
		newCreateTrackCmd(svc, cwd, stdout),
	)
	return cmd
}

func newCreateTaskCmd(svc app.App, cwd string, stdin io.Reader, stdout io.Writer) *cobra.Command {
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
			bodyBytes, err := readBody(stdin, body)
			if err != nil {
				return err
			}
			spec := app.TaskSpec{Title: args[0], Slug: slug, Section: section, BlockedBy: blockedBy, Body: bodyBytes}
			id, path, err := svc.CreateTask(app.Scope{Cwd: cwd, In: in}, spec)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "%s\t%s\n", id, path)
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

func readBody(stdin io.Reader, body bool) ([]byte, error) {
	if !body {
		return nil, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	return data, nil
}

func newCreateTrackCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
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
			path, err := svc.CreateTrack(app.Scope{Cwd: cwd, In: in}, args[0], title)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "%s\t%s\n", args[0], path)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "track title (default: the name)")
	addInFlag(cmd, &in)
	return cmd
}

// Package cli is duty's presentation layer: thin cobra commands in a
// kubectl-style verb → resource grammar that parse flags, delegate to the
// app services, and format output. Cobra's own error and usage printing is
// silenced to keep the contract: quiet on success, one lowercase stderr line
// per error, exit 0/1, and 2 on a missing or unknown command.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/fsys"
)

// missingCommandError reports an invocation naming no command or resource;
// it maps to exit 2.
type missingCommandError string

// Error renders the one-line usage message.
func (e missingCommandError) Error() string { return string(e) }

// unknownCommandError names a command Run does not know; it maps to exit 2.
type unknownCommandError string

// Error renders the one-line unknown-command message.
func (e unknownCommandError) Error() string {
	return fmt.Sprintf("unknown command %q", string(e))
}

// errNoCommand reports an invocation naming no command at all.
var errNoCommand = missingCommandError("usage: duty <command> [args]")

// Run executes one duty command over the real filesystem. args is the command
// line without the program name; stdin feeds commands that read input; stdout
// receives command output; stderr receives one-line error messages. It returns
// the process exit code: 0 on success, 2 on a missing or unknown command, 1 on
// any other error.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, errNoCommand)
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	root := newRoot(cwd, stdin, stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		var missing missingCommandError
		var unknown unknownCommandError
		if errors.As(err, &missing) || errors.As(err, &unknown) {
			return 2
		}
		return 1
	}
	return 0
}

// newRoot assembles the duty command tree over the real filesystem, rooted
// at cwd, with cobra's own error and usage printing silenced.
func newRoot(cwd string, stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	var f fsys.FS = fsys.OS{}
	a := app.New(f)
	root := &cobra.Command{
		Use:           "duty <command> [args]",
		Short:         "file-based task system: markdown tasks + board indexes",
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errNoCommand
			}
			return unknownCommandError(args[0])
		},
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		newInitCmd(a, cwd),
		newCreateCmd(a, cwd, stdout),
		newGetCmd(a, cwd, stdout),
		newListCmd(a, cwd, stdout),
		newStatusCmd(a, cwd),
		newReportCmd(a, cwd, stdin),
		newMoveCmd(a, cwd),
		newArchiveCmd(a, cwd),
		newDeleteCmd(a, cwd),
		newTUICmd(f, cwd),
	)
	return root
}

// newGroupCmd builds a verb command that only dispatches to its resource
// subcommands: invoked bare it reports usage, with an unknown resource it
// reports the unknown command — both map to exit 2.
func newGroupCmd(use, short, usage string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return missingCommandError(usage)
			}
			return unknownCommandError(args[0])
		},
	}
}

// stringList is a repeatable string flag; each occurrence may also carry
// several comma-separated values.
type stringList []string

// String renders the collected values, satisfying pflag.Value.
func (l *stringList) String() string { return strings.Join(*l, ",") }

// Set appends the comma-separated values in v, satisfying pflag.Value.
func (l *stringList) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*l = append(*l, s)
		}
	}
	return nil
}

// Type names the flag's value in help output, satisfying pflag.Value.
func (l *stringList) Type() string { return "id" }

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
	"github.com/raphaelCamblong/duty/internal/fetch"
	"github.com/raphaelCamblong/duty/internal/fsys"
)

// missingCommandError reports an invocation naming no command or resource;
// it maps to exit 2.
type missingCommandError string

// Error renders the one-line usage message.
func (e missingCommandError) Error() string { return string(e) }

// unknownCommandError names a command Run does not know, with an optional
// typo suggestion; it maps to exit 2.
type unknownCommandError struct {
	name       string
	suggestion string
}

// Error renders the one-line unknown-command message, with a "did you mean"
// hint when a close match exists.
func (e unknownCommandError) Error() string {
	if e.suggestion == "" {
		return fmt.Sprintf("unknown command %q", e.name)
	}
	return fmt.Sprintf("unknown command %q — did you mean %q?", e.name, e.suggestion)
}

// errNoCommand reports an invocation naming no command at all.
var errNoCommand = missingCommandError("usage: duty <command> [args]")

// unknownCommand builds the unknown-command error for name, suggesting the
// closest of cmd's subcommands when one is within SuggestionsMinimumDistance.
func unknownCommand(cmd *cobra.Command, name string) error {
	suggestion := ""
	if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
		suggestion = suggestions[0]
	}
	return unknownCommandError{name: name, suggestion: suggestion}
}

// Run executes one duty command over the real filesystem. args is the command
// line without the program name; stdin feeds commands that read input; stdout
// receives command output; stderr receives one-line error messages. It returns
// the process exit code: 0 on success, 2 on a missing or unknown command, 1 on
// any other error.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, errNoCommand)
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	root := newRoot(cwd, stdin, stdout, stderr, version)
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

// Command group ids, shared between the root command's grouping and each
// subcommand's assignment.
const (
	groupAuthor    = "author"
	groupWork      = "work"
	groupRead      = "read"
	groupInterface = "interface"
)

// rootLong is the root command's long help: what duty is, then the five-step
// lifecycle it drives.
const rootLong = `duty is a file-based task system: markdown task files plus nested board
indexes, kept in sync by one binary.

The lifecycle:
  duty get next                  find the next actionable task
  duty status <id> in-progress   start it
  tick gate checkboxes in the task file as they pass
  duty status <id> done          then duty report <id> with what changed
  duty archive                   move every done task into its board's archive/`

// rootExample is a copy-pasteable run through the lifecycle above.
const rootExample = `  duty get next --agent
  duty status T-07 in-progress
  duty status T-07 done
  duty report T-07 < report.txt
  duty archive`

// newRoot assembles the duty command tree over the real filesystem, rooted
// at cwd, with cobra's own error and usage printing silenced.
func newRoot(cwd string, stdin io.Reader, stdout, stderr io.Writer, version string) *cobra.Command {
	root := rootCmd(version)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	addCommands(root, cwd, stdin, stdout)
	return root
}

// rootCmd builds the bare root command: identity, help text, and cobra's own
// error and usage printing silenced.
func rootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "duty <command> [args]",
		Short:   "file-based task system: markdown tasks + board indexes",
		Long:    rootLong,
		Example: rootExample,
		Version: version,
	}
	dispatchOnly(cmd, errNoCommand)
	return cmd
}

// dispatchOnly turns cmd into a pure dispatcher: arbitrary args, cobra's own
// error and usage printing silenced, and a RunE that returns missing when
// invoked bare or the unknown-command error otherwise — both mapping to exit 2.
func dispatchOnly(cmd *cobra.Command, missing error) {
	cmd.Args = cobra.ArbitraryArgs
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SuggestionsMinimumDistance = 2
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return missing
		}
		return unknownCommand(cmd, args[0])
	}
}

// addCommands registers the help groups and every subcommand under root,
// wired over the real filesystem rooted at cwd.
func addCommands(root *cobra.Command, cwd string, stdin io.Reader, stdout io.Writer) {
	var f fsys.FS = fsys.OS{}
	a := app.New(f)
	home, _ := os.UserHomeDir()
	root.SetCompletionCommandGroupID(groupInterface)
	root.AddGroup(
		&cobra.Group{ID: groupAuthor, Title: "Author Commands:"},
		&cobra.Group{ID: groupWork, Title: "Work Commands:"},
		&cobra.Group{ID: groupRead, Title: "Read Commands:"},
		&cobra.Group{ID: groupInterface, Title: "Interface Commands:"},
	)
	root.AddCommand(
		grouped(newInitCmd(a, cwd, stdout), groupAuthor),
		grouped(newCreateCmd(a, cwd, stdin, stdout), groupAuthor),
		grouped(newSetCmd(a, cwd, stdin), groupAuthor),
		grouped(newGetCmd(a, cwd, stdout), groupRead),
		newListCmd(a, cwd, stdout),
		grouped(newStatusCmd(a, cwd), groupWork),
		grouped(newReportCmd(a, cwd, stdin), groupWork),
		grouped(newGatesCmd(a, cwd, stdout), groupWork),
		grouped(newMoveCmd(a, cwd), groupWork),
		grouped(newArchiveCmd(a, cwd), groupWork),
		grouped(newDeleteCmd(a, cwd), groupWork),
		grouped(newTUICmd(f, cwd), groupInterface),
		grouped(newWatchCmd(a, f, cwd, stdout), groupInterface),
		grouped(newSkillCmd(a, fetch.HTTP{}, cwd, home, stdout), groupInterface),
	)
}

// grouped assigns cmd to the help group id and returns it.
func grouped(cmd *cobra.Command, id string) *cobra.Command {
	cmd.GroupID = id
	return cmd
}

// claimer resolves the agent name a claim records: the --as flag value when
// set, else $DUTY_AGENT, else empty for an unnamed claim.
func claimer(as string) string {
	if as = strings.TrimSpace(as); as != "" {
		return as
	}
	return strings.TrimSpace(os.Getenv("DUTY_AGENT"))
}

// addAsFlag registers the --as flag on cmd, binding it to as: the agent name a
// claim records. Shared by every command that can move a task to in-progress.
func addAsFlag(cmd *cobra.Command, as *string) {
	cmd.Flags().StringVar(as, "as", "", "agent name to record as the claimer (falls back to $DUTY_AGENT)")
}

// newGroupCmd builds a verb command that only dispatches to its resource
// subcommands: invoked bare it reports usage, with an unknown resource it
// reports the unknown command — both map to exit 2.
func newGroupCmd(use, short, usage, example string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
	}
	dispatchOnly(cmd, missingCommandError(usage))
	return cmd
}

// addInFlag registers the local --in flag on cmd, binding it to in: a
// root-relative track path ("." = root board) selecting the board the command
// acts on instead of the one derived from cwd. Shared by every board-scoped
// command.
func addInFlag(cmd *cobra.Command, in *string) {
	cmd.Flags().StringVar(in, "in", "", `board to act on by track path from the tree root ("." = root)`)
}

// tsv joins fields into one agent-output record; the tab is the wire contract
// for --agent output.
func tsv(fields ...string) string { return strings.Join(fields, "\t") }

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

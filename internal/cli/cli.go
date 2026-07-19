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

// missingCommandError marks a bare invocation; Run maps it to exit 2.
type missingCommandError string

func (e missingCommandError) Error() string { return string(e) }

// unknownCommandError names an unrecognized command; Run maps it to exit 2.
type unknownCommandError struct {
	name       string
	suggestion string
}

func (e unknownCommandError) Error() string {
	if e.suggestion == "" {
		return fmt.Sprintf("unknown command %q", e.name)
	}
	return fmt.Sprintf("unknown command %q — did you mean %q?", e.name, e.suggestion)
}

var errNoCommand = missingCommandError("usage: duty <command> [args]")

func unknownCommand(cmd *cobra.Command, name string) error {
	suggestion := ""
	if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
		suggestion = suggestions[0]
	}
	return unknownCommandError{name: name, suggestion: suggestion}
}

// Run executes one duty command, returning the exit code: 0 ok, 2 missing/unknown command, 1 otherwise.
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

const (
	groupAuthor    = "author"
	groupWork      = "work"
	groupRead      = "read"
	groupInterface = "interface"
)

const rootLong = `duty is a file-based task system: markdown task files plus nested board
indexes, kept in sync by one binary.

The lifecycle:
  duty get next                  find the next actionable task
  duty status <id> in-progress   start it
  tick gate checkboxes in the task file as they pass
  duty status <id> done          then duty report <id> with what changed
  duty archive                   move every done task into its board's archive/`

const rootExample = `  duty get next --agent
  duty status T-07 in-progress
  duty status T-07 done
  duty report T-07 < report.txt
  duty archive`

func newRoot(cwd string, stdin io.Reader, stdout, stderr io.Writer, version string) *cobra.Command {
	root := rootCmd(version)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	addCommands(root, cwd, stdin, stdout)
	return root
}

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

func addCommands(root *cobra.Command, cwd string, stdin io.Reader, stdout io.Writer) {
	var fs fsys.FS = fsys.OS{}
	svc := app.New(fs)
	home, _ := os.UserHomeDir()
	root.SetCompletionCommandGroupID(groupInterface)
	root.AddGroup(
		&cobra.Group{ID: groupAuthor, Title: "Author Commands:"},
		&cobra.Group{ID: groupWork, Title: "Work Commands:"},
		&cobra.Group{ID: groupRead, Title: "Read Commands:"},
		&cobra.Group{ID: groupInterface, Title: "Interface Commands:"},
	)
	root.AddCommand(
		grouped(newInitCmd(svc, cwd, stdout), groupAuthor),
		grouped(newCreateCmd(svc, cwd, stdin, stdout), groupAuthor),
		grouped(newSetCmd(svc, cwd, stdin), groupAuthor),
		grouped(newGetCmd(svc, cwd, stdout), groupRead),
		newListCmd(svc, cwd, stdout),
		grouped(newStatusCmd(svc, cwd), groupWork),
		grouped(newReportCmd(svc, cwd, stdin), groupWork),
		grouped(newGatesCmd(svc, cwd, stdout), groupWork),
		grouped(newMoveCmd(svc, cwd), groupWork),
		grouped(newArchiveCmd(svc, cwd), groupWork),
		grouped(newDeleteCmd(svc, cwd), groupWork),
		grouped(newTUICmd(fs, cwd), groupInterface),
		grouped(newWatchCmd(svc, fs, cwd, stdout), groupInterface),
		grouped(newSkillCmd(skillCtx{app: svc, fetcher: fetch.HTTP{}, cwd: cwd, home: home, out: stdout}), groupInterface),
	)
}

func grouped(cmd *cobra.Command, id string) *cobra.Command {
	cmd.GroupID = id
	return cmd
}

func claimer(as string) string {
	if as = strings.TrimSpace(as); as != "" {
		return as
	}
	return strings.TrimSpace(os.Getenv("DUTY_AGENT"))
}

func addAsFlag(cmd *cobra.Command, as *string) {
	cmd.Flags().StringVar(as, "as", "", "agent name to record as the claimer (falls back to $DUTY_AGENT)")
}

func newGroupCmd(use, short, usage, example string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
	}
	dispatchOnly(cmd, missingCommandError(usage))
	return cmd
}

func addInFlag(cmd *cobra.Command, in *string) {
	cmd.Flags().StringVar(in, "in", "", `board to act on by track path from the tree root ("." = root)`)
}

// tsv joins fields with tab, the wire contract for --agent output.
func tsv(fields ...string) string { return strings.Join(fields, "\t") }

// stringList is a repeatable, comma-splitting flag implementing pflag.Value.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			*l = append(*l, item)
		}
	}
	return nil
}

func (l *stringList) Type() string { return "id" }

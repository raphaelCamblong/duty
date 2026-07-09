// Package cli dispatches duty's subcommands: one flag.FlagSet per command,
// no framework. The sync invariant lives here — every mutating handler edits
// the task file AND its board row in the same command — and every write goes
// through fsutil.WriteAtomic. Commands are quiet on success; errors are one
// lowercase line on stderr and a non-zero exit code.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// boardFile is the index file every board directory holds.
const boardFile = "BOARD.md"

// nameRE validates sub-board folder names and task filename slugs.
var nameRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// Run executes one duty command. args is the command line without the program
// name; stdin feeds commands that read input; stdout receives command output;
// stderr receives one-line error messages. It returns the process exit code:
// 0 on success, 2 on a missing or unknown command, 1 on any other error.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: duty <command> [args]")
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch cmd := args[0]; cmd {
	case "init":
		err = runInit(cwd, args[1:])
	case "create":
		err = runCreate(cwd, args[1:], stdout)
	case "board":
		err = runBoard(cwd, args[1:])
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// positionals parses args against fs and returns the positional arguments.
// Flags may appear before, between, and after positionals (the spec's usage
// lines put the title first, flags after). A help request surfaces as an
// error carrying the command's usage line.
func positionals(fs *flag.FlagSet, args []string, usage string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, errors.New(usage)
			}
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// stringList is a repeatable string flag; each occurrence may also carry
// several comma-separated values.
type stringList []string

// String renders the collected values, satisfying flag.Value.
func (l *stringList) String() string { return strings.Join(*l, ",") }

// Set appends the comma-separated values in v, satisfying flag.Value.
func (l *stringList) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*l = append(*l, s)
		}
	}
	return nil
}

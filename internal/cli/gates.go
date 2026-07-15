package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/task"
)

const (
	gatesUsage      = "usage: duty gates <id> [list]"
	gatesAddUsage   = "usage: duty gates add <id> <text>"
	gatesExample    = "  duty gates T-07\n  duty gates add T-07 \"build passes\"\n  duty gates check T-07 1"
	gatesFlipUsage  = "usage: duty gates %s <id> <n>"
	gatesAddExample = `  duty gates add T-07 "build passes"`
)

// newGatesCmd builds the gates command: list a task's gates (bare or with the
// "list" word), or add, check, and uncheck them.
func newGatesCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var agent bool
	cmd := &cobra.Command{
		Use:     "gates <id>",
		Short:   "list a task's gates, or add/check/uncheck them",
		Example: gatesExample,
		RunE: func(_ *cobra.Command, args []string) error {
			id, err := gatesListID(args)
			if err != nil {
				return err
			}
			gates, err := a.Gates(cwd, id)
			if err != nil {
				return err
			}
			printGates(stdout, gates, agent)
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: index, done, text")
	cmd.AddCommand(
		newGatesAddCmd(a, cwd),
		newGatesFlipCmd(a, cwd, "check", true),
		newGatesFlipCmd(a, cwd, "uncheck", false),
	)
	return cmd
}

// gatesListID extracts the task id from a gates-list invocation: "<id>" or the
// explicit "<id> list".
func gatesListID(args []string) (string, error) {
	if len(args) == 1 && args[0] != "" {
		return args[0], nil
	}
	if len(args) == 2 && args[0] != "" && args[1] == "list" {
		return args[0], nil
	}
	return "", errors.New(gatesUsage)
}

// newGatesAddCmd builds gates add: append a gate to a task.
func newGatesAddCmd(a app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:     "add <id> <text>",
		Short:   "append a gate to a task",
		Example: gatesAddExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return errors.New(gatesAddUsage)
			}
			return a.AddGate(cwd, args[0], args[1])
		},
	}
}

// newGatesFlipCmd builds gates check / uncheck: tick or untick a task's n-th
// gate (1-based).
func newGatesFlipCmd(a app.App, cwd, verb string, done bool) *cobra.Command {
	usage := fmt.Sprintf(gatesFlipUsage, verb)
	return &cobra.Command{
		Use:     verb + " <id> <n>",
		Short:   "tick or untick a task's n-th gate",
		Example: "  duty gates " + verb + " T-07 1",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" {
				return errors.New(usage)
			}
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				return errors.New(usage)
			}
			return a.SetGate(cwd, args[0], n, done)
		},
	}
}

// printGates writes gates 1-based: human "1 [x] text", or with agent a TSV
// record "index<TAB>done<TAB>text" (done "true"/"false").
func printGates(w io.Writer, gates []task.Gate, agent bool) {
	for i, g := range gates {
		if agent {
			fmt.Fprintf(w, "%d\t%s\t%s\n", i+1, strconv.FormatBool(g.Done), g.Text)
			continue
		}
		fmt.Fprintf(w, "%d %s %s\n", i+1, checkbox(g.Done), g.Text)
	}
}

// checkbox renders a gate's ticked state as a markdown checkbox.
func checkbox(done bool) string {
	if done {
		return "[x]"
	}
	return "[ ]"
}

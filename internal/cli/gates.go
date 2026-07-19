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
	gatesAddUsage   = "usage: duty gates add <id> <text> [<text>...]"
	gatesExample    = "  duty gates T-07\n  duty gates add T-07 \"build passes\" \"tests green\"\n  duty gates check T-07 --all"
	gatesFlipUsage  = "usage: duty gates %s <id> <n> (or --all)"
	gatesAddExample = `  duty gates add T-07 "build passes" "tests green"`
)

func newGatesCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
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
			gates, err := svc.Gates(cwd, id)
			if err != nil {
				return err
			}
			printGates(stdout, gates, agent)
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: index, done, text")
	cmd.AddCommand(
		newGatesAddCmd(svc, cwd),
		newGatesFlipCmd(svc, cwd, "check", true),
		newGatesFlipCmd(svc, cwd, "uncheck", false),
	)
	return cmd
}

func gatesListID(args []string) (string, error) {
	if len(args) == 1 && args[0] != "" {
		return args[0], nil
	}
	if len(args) == 2 && args[0] != "" && args[1] == "list" {
		return args[0], nil
	}
	return "", errors.New(gatesUsage)
}

func newGatesAddCmd(svc app.App, cwd string) *cobra.Command {
	return &cobra.Command{
		Use:     "add <id> <text>...",
		Short:   "append one or more gates to a task",
		Example: gatesAddExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) < 2 || args[0] == "" || hasEmpty(args[1:]) {
				return errors.New(gatesAddUsage)
			}
			return svc.AddGates(cwd, args[0], args[1:])
		},
	}
}

func newGatesFlipCmd(svc app.App, cwd, verb string, done bool) *cobra.Command {
	usage := fmt.Sprintf(gatesFlipUsage, verb)
	var all bool
	cmd := &cobra.Command{
		Use:     verb + " <id> <n>",
		Short:   "tick or untick a task's n-th gate, or --all of them",
		Example: "  duty gates " + verb + " T-07 1\n  duty gates " + verb + " T-07 --all",
		RunE: func(_ *cobra.Command, args []string) error {
			if all {
				if len(args) != 1 || args[0] == "" {
					return errors.New(usage)
				}
				return svc.SetAllGates(cwd, args[0], done)
			}
			if len(args) != 2 || args[0] == "" {
				return errors.New(usage)
			}
			index, err := strconv.Atoi(args[1])
			if err != nil || index < 1 {
				return errors.New(usage)
			}
			return svc.SetGate(cwd, args[0], index, done)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "flip every gate in one write")
	return cmd
}

func hasEmpty(ss []string) bool {
	for _, text := range ss {
		if text == "" {
			return true
		}
	}
	return false
}

// printGates numbers gates 1-based, plain text or --agent TSV (index, done, text).
func printGates(out io.Writer, gates []task.Gate, agent bool) {
	for i, gate := range gates {
		if agent {
			fmt.Fprintf(out, "%d\t%s\t%s\n", i+1, strconv.FormatBool(gate.Done), gate.Text)
			continue
		}
		fmt.Fprintf(out, "%d %s %s\n", i+1, checkbox(gate.Done), gate.Text)
	}
}

func checkbox(done bool) string {
	if done {
		return "[x]"
	}
	return "[ ]"
}

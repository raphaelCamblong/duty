package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const listUsage = "usage: duty list [--status S] [--agent]"

// newListCmd builds the list command: every open task in the current board
// and below, one human line or one --agent TSV record per task.
func newListCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var (
		status string
		agent  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list open tasks from the files, with drift flags",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(listUsage)
			}
			rows, err := a.List(cwd, status)
			if err != nil {
				return err
			}
			for _, r := range rows {
				if agent {
					fmt.Fprintln(stdout, tsvLine(r))
					continue
				}
				fmt.Fprintln(stdout, humanLine(r))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "list only this status")
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, board-path, status, title, drift")
	return cmd
}

// humanLine renders r for human reading: "[board/ ]id  status  title[  drift]".
func humanLine(r app.Row) string {
	var b strings.Builder
	if r.Board != "." {
		b.WriteString(r.Board)
		b.WriteString("/ ")
	}
	b.WriteString(r.ID)
	b.WriteString("  ")
	b.WriteString(r.Status)
	b.WriteString("  ")
	b.WriteString(r.Title)
	if drift := humanDrift(r); drift != "" {
		b.WriteString("  ")
		b.WriteString(drift)
	}
	return b.String()
}

// humanDrift renders r's drift flag for a human, "" when in sync.
func humanDrift(r app.Row) string {
	switch {
	case r.RowMissing:
		return "⚠ board says missing"
	case r.RowStatus != "":
		return "⚠ board says " + r.RowStatus
	}
	return ""
}

// tsvLine renders r as one agent-output record:
// id<TAB>board-path<TAB>status<TAB>title<TAB>drift.
func tsvLine(r app.Row) string {
	return strings.Join([]string{r.ID, r.Board, r.Status, r.Title, agentDrift(r)}, "\t")
}

// agentDrift renders r's drift flag for --agent output: "", "board=<status>",
// or "no-row".
func agentDrift(r app.Row) string {
	switch {
	case r.RowMissing:
		return "no-row"
	case r.RowStatus != "":
		return "board=" + r.RowStatus
	}
	return ""
}

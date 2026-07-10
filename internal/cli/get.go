package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
)

const (
	getUsage       = "usage: duty get <task|tasks|tracks|next> [args]"
	getTaskUsage   = "usage: duty get task <id> [--agent]"
	getTasksUsage  = "usage: duty get tasks [--status S] [--agent]"
	getTracksUsage = "usage: duty get tracks [--agent]"
	getNextUsage   = "usage: duty get next [--agent]"
)

// taskKeyWidth pads the key column of get task's human output; it is the
// width of the widest key including its colon ("blocked-by:").
const taskKeyWidth = len("blocked-by:")

// newGetCmd builds the get verb: resource subcommands for reading state.
func newGetCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	cmd := newGroupCmd("get", "read tasks and tracks from the files", getUsage)
	cmd.AddCommand(
		newGetTaskCmd(a, cwd, stdout),
		newGetTasksCmd(a, cwd, stdout, "tasks", false),
		newGetTracksCmd(a, cwd, stdout),
		newGetNextCmd(a, cwd, stdout),
	)
	return cmd
}

// newListCmd builds the hidden top-level list alias for get tasks.
func newListCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	return newGetTasksCmd(a, cwd, stdout, "list", true)
}

// newGetTaskCmd builds get task: one task's metadata and file path, human
// aligned or a single --agent TSV record.
func newGetTaskCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var agent bool
	cmd := &cobra.Command{
		Use:   "task <id>",
		Short: "show one task's metadata and file path",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(getTaskUsage)
			}
			info, err := a.GetTask(cwd, args[0])
			if err != nil {
				return err
			}
			if agent {
				fmt.Fprintln(stdout, taskAgent(info))
				return nil
			}
			fmt.Fprintln(stdout, taskHuman(info))
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, track, status, title, gates-done, gates-total, blocked-by, path")
	return cmd
}

// newGetTracksCmd builds get tracks: one line per board — the root included —
// with its title and per-status own-task counts.
func newGetTracksCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var agent bool
	cmd := &cobra.Command{
		Use:   "tracks",
		Short: "show every board's per-status task counts",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getTracksUsage)
			}
			tracks, err := a.GetTracks(cwd)
			if err != nil {
				return err
			}
			if agent {
				for _, tr := range tracks {
					fmt.Fprintln(stdout, trackAgent(tr))
				}
				return nil
			}
			for _, line := range tracksHuman(tracks) {
				fmt.Fprintln(stdout, line)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: path, title, todo, in-progress, done, blocked, archived")
	return cmd
}

// newGetNextCmd builds get next: the first actionable task, or no output when
// nothing is ready.
func newGetNextCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	var agent bool
	cmd := &cobra.Command{
		Use:   "next",
		Short: "show the first actionable task (empty when nothing is ready)",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getNextUsage)
			}
			info, err := a.GetNext(cwd)
			if err != nil {
				return err
			}
			if info == nil {
				return nil
			}
			if agent {
				fmt.Fprintln(stdout, taskAgent(*info))
				return nil
			}
			fmt.Fprintln(stdout, taskHuman(*info))
			return nil
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output, same fields as get task")
	return cmd
}

// newGetTasksCmd builds the tasks reader under the given name: every open
// task in the current board and below, one human line or one --agent TSV
// record per task.
func newGetTasksCmd(a app.App, cwd string, stdout io.Writer, use string, hidden bool) *cobra.Command {
	var (
		status string
		agent  bool
	)
	cmd := &cobra.Command{
		Use:    use,
		Short:  "list open tasks from the files, with drift flags",
		Hidden: hidden,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getTasksUsage)
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

// taskHuman renders info as aligned "key: value" lines: id, title, status,
// track, blocked-by, gates n/m, path.
func taskHuman(info app.TaskInfo) string {
	var b strings.Builder
	kv(&b, "id", info.ID)
	kv(&b, "title", info.Title)
	kv(&b, "status", info.Status)
	kv(&b, "track", info.Track)
	kv(&b, "blocked-by", blockedByHuman(info.BlockedBy))
	kv(&b, "gates", fmt.Sprintf("%d/%d", info.GatesDone, info.GatesTotal))
	kv(&b, "path", info.Path)
	return strings.TrimRight(b.String(), "\n")
}

// kv writes one "key:<pad>value" line, the pad aligning every value to the
// widest key.
func kv(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%-*s %s\n", taskKeyWidth, key+":", value)
}

// blockedByHuman joins ids with ", ", or "none" when the list is empty.
func blockedByHuman(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

// taskAgent renders info as one agent-output TSV record: id, track, status,
// title, gates-done, gates-total, blocked-by (comma-joined), path.
func taskAgent(info app.TaskInfo) string {
	return strings.Join([]string{
		info.ID,
		info.Track,
		info.Status,
		info.Title,
		strconv.Itoa(info.GatesDone),
		strconv.Itoa(info.GatesTotal),
		strings.Join(info.BlockedBy, ","),
		info.Path,
	}, "\t")
}

// tracksHuman renders tracks as aligned columns: path, title, then the
// per-status counts and archived count in fixed order.
func tracksHuman(tracks []app.TrackInfo) []string {
	pathW, titleW := 0, 0
	for _, tr := range tracks {
		if len(tr.Path) > pathW {
			pathW = len(tr.Path)
		}
		if len(tr.Title) > titleW {
			titleW = len(tr.Title)
		}
	}
	lines := make([]string, 0, len(tracks))
	for _, tr := range tracks {
		counts := fmt.Sprintf("%d todo · %d in-progress · %d done · %d blocked · %d archived",
			tr.Todo, tr.InProgress, tr.Done, tr.Blocked, tr.Archived)
		lines = append(lines, fmt.Sprintf("%-*s  %-*s  %s", pathW, tr.Path, titleW, tr.Title, counts))
	}
	return lines
}

// trackAgent renders one track as a TSV record: path, title, todo,
// in-progress, done, blocked, archived — fixed column order is the contract.
func trackAgent(tr app.TrackInfo) string {
	return strings.Join([]string{
		tr.Path,
		tr.Title,
		strconv.Itoa(tr.Todo),
		strconv.Itoa(tr.InProgress),
		strconv.Itoa(tr.Done),
		strconv.Itoa(tr.Blocked),
		strconv.Itoa(tr.Archived),
	}, "\t")
}

// humanLine renders r for human reading: "[track/ ]id  status  title[  drift]".
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

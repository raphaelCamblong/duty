package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/humanize"
	"github.com/raphaelCamblong/duty/internal/task"
)

// ageStyle dims the age and waits columns in human output.
var ageStyle = lipgloss.NewStyle().Faint(true)

const (
	getUsage       = "usage: duty get <task|tasks|tracks|next> [args]"
	getTaskUsage   = "usage: duty get task <id> [--agent] [--section NAME] [--body]"
	getTasksUsage  = "usage: duty get tasks [--status S] [--agent]"
	getTracksUsage = "usage: duty get tracks [--agent]"
	getNextUsage   = "usage: duty get next [--claim] [--agent]"

	getExample = `  duty get next --agent
  duty get task T-07`
	getTaskExample   = `  duty get task T-07`
	getTracksExample = `  duty get tracks --agent`
	getNextExample   = `  duty get next --agent
  duty get next --claim`
)

// taskKeyWidth pads kv's keys to the width of the widest, "blocked-by:".
const taskKeyWidth = len("blocked-by:")

func newGetCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
	cmd := newGroupCmd("get", "read tasks and tracks from the files", getUsage, getExample)
	cmd.AddCommand(
		newGetTaskCmd(svc, cwd, stdout),
		newGetTasksCmd(svc, cwd, stdout, "tasks", false),
		newGetTracksCmd(svc, cwd, stdout),
		newGetNextCmd(svc, cwd, stdout),
	)
	return cmd
}

func newListCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
	return newGetTasksCmd(svc, cwd, stdout, "list", true)
}

func newGetTaskCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
	var (
		agent   bool
		section string
		body    bool
	)
	cmd := &cobra.Command{
		Use:     "task <id>",
		Short:   "show one task's metadata and file path",
		Example: getTaskExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return errors.New(getTaskUsage)
			}
			return getTaskOut(svc, cwd, taskQuery{id: args[0], section: section, body: body, agent: agent}, stdout)
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, track, status, title, gates-done, gates-total, blocked-by, path, updated, claimed-by")
	cmd.Flags().StringVar(&section, "section", "", "print only this section's body")
	cmd.Flags().BoolVar(&body, "body", false, "print the whole body below the frontmatter")
	cmd.MarkFlagsMutuallyExclusive("body", "section")
	cmd.MarkFlagsMutuallyExclusive("body", "agent")
	return cmd
}

// taskQuery is one `get task` request: an id and how to render it.
type taskQuery struct {
	id      string
	section string
	body    bool
	agent   bool
}

func getTaskOut(svc app.App, cwd string, query taskQuery, stdout io.Writer) error {
	if query.body {
		text, err := svc.Body(cwd, query.id)
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, text)
		return nil
	}
	if query.section != "" {
		sec, err := svc.Section(cwd, query.id, query.section)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, sec)
		return nil
	}
	info, err := svc.GetTask(cwd, query.id)
	if err != nil {
		return err
	}
	if query.agent {
		fmt.Fprintln(stdout, taskAgent(info))
		return nil
	}
	fmt.Fprintln(stdout, taskHuman(info))
	return nil
}

func newGetTracksCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
	var (
		agent bool
		in    string
	)
	cmd := &cobra.Command{
		Use:     "tracks",
		Short:   "show every board's per-status task counts",
		Example: getTracksExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getTracksUsage)
			}
			tracks, err := svc.GetTracks(app.Scope{Cwd: cwd, In: in})
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
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: path, title, todo, in-progress, done, blocked, archived, backlog")
	addInFlag(cmd, &in)
	return cmd
}

// newGetNextCmd's --claim marks the task in-progress so parallel agents get distinct tasks.
func newGetNextCmd(svc app.App, cwd string, stdout io.Writer) *cobra.Command {
	var (
		agent bool
		claim bool
		in    string
		as    string
	)
	cmd := &cobra.Command{
		Use:     "next",
		Short:   "show the first actionable task (empty when nothing is ready)",
		Example: getNextExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getNextUsage)
			}
			info, err := svc.GetNext(app.Scope{Cwd: cwd, In: in}, claim, claimer(as))
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
	cmd.Flags().BoolVar(&claim, "claim", false, "atomically mark the task in-progress and print it")
	addInFlag(cmd, &in)
	addAsFlag(cmd, &as)
	return cmd
}

func newGetTasksCmd(svc app.App, cwd string, stdout io.Writer, use string, hidden bool) *cobra.Command {
	var (
		status string
		agent  bool
		in     string
	)
	cmd := &cobra.Command{
		Use:     use,
		Short:   "list open tasks from the files, with drift flags",
		Example: fmt.Sprintf("  duty %s --status todo", use),
		Hidden:  hidden,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(getTasksUsage)
			}
			rows, err := svc.List(app.Scope{Cwd: cwd, In: in}, status)
			if err != nil {
				return err
			}
			for _, row := range rows {
				if agent {
					fmt.Fprintln(stdout, tsvLine(row))
					continue
				}
				fmt.Fprintln(stdout, humanLine(row))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "list only this status")
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, board-path, status, title, drift, updated, claimed-by, waits")
	addInFlag(cmd, &in)
	return cmd
}

func taskHuman(info app.TaskInfo) string {
	var builder strings.Builder
	kv(&builder, "id", info.ID)
	kv(&builder, "title", info.Title)
	kv(&builder, "status", info.Status)
	if info.ClaimedBy != "" {
		kv(&builder, "claimed-by", info.ClaimedBy)
	}
	kv(&builder, "track", info.Track)
	kv(&builder, "blocked-by", blockedByHuman(info.Deps))
	kv(&builder, "gates", fmt.Sprintf("%d/%d", info.GatesDone, info.GatesTotal))
	kv(&builder, "updated", humanize.RelTime(info.UpdatedAt, time.Now()))
	kv(&builder, "path", info.Path)
	return strings.TrimRight(builder.String(), "\n")
}

func kv(builder *strings.Builder, key, value string) {
	fmt.Fprintf(builder, "%-*s %s\n", taskKeyWidth, key+":", value)
}

func blockedByHuman(deps []app.Dep) string {
	if len(deps) == 0 {
		return "none"
	}
	parts := make([]string, len(deps))
	for i, dep := range deps {
		parts[i] = dep.ID + " (" + dep.Status + ")"
	}
	return strings.Join(parts, ", ")
}

// taskAgent's TSV column order is a wire contract: new fields only ever append.
func taskAgent(info app.TaskInfo) string {
	return tsv(
		info.ID,
		info.Track,
		info.Status,
		info.Title,
		strconv.Itoa(info.GatesDone),
		strconv.Itoa(info.GatesTotal),
		strings.Join(info.BlockedBy, ","),
		info.Path,
		info.UpdatedAt.Format(time.RFC3339),
		info.ClaimedBy,
	)
}

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
		counts := fmt.Sprintf("%d todo · %d in-progress · %d done · %d blocked · %d backlog · %d archived",
			tr.Todo, tr.InProgress, tr.Done, tr.Blocked, tr.Backlog, tr.Archived)
		lines = append(lines, fmt.Sprintf("%-*s  %-*s  %s", pathW, tr.Path, titleW, tr.Title, counts))
	}
	return lines
}

// trackAgent's column order is a wire contract: new fields append, so backlog trails archived.
func trackAgent(tr app.TrackInfo) string {
	return tsv(
		tr.Path,
		tr.Title,
		strconv.Itoa(tr.Todo),
		strconv.Itoa(tr.InProgress),
		strconv.Itoa(tr.Done),
		strconv.Itoa(tr.Blocked),
		strconv.Itoa(tr.Archived),
		strconv.Itoa(tr.Backlog),
	)
}

// humanLine renders "[track/ ]id  status  title[  drift]  age[  waits]".
func humanLine(row app.Row) string {
	var builder strings.Builder
	if row.Board != "." {
		builder.WriteString(row.Board)
		builder.WriteString("/ ")
	}
	builder.WriteString(row.ID)
	builder.WriteString("  ")
	builder.WriteString(statusCell(row))
	builder.WriteString("  ")
	builder.WriteString(row.Title)
	if drift := humanDrift(row); drift != "" {
		builder.WriteString("  ")
		builder.WriteString(drift)
	}
	builder.WriteString("  ")
	builder.WriteString(ageStyle.Render(humanize.RelTime(row.UpdatedAt, time.Now())))
	if wait := waitsCell(row); wait != "" {
		builder.WriteString("  ")
		builder.WriteString(ageStyle.Render(wait))
	}
	return builder.String()
}

func waitsCell(row app.Row) string {
	if len(row.Waits) == 0 {
		return ""
	}
	return "waits " + strings.Join(row.Waits, ",")
}

func statusCell(row app.Row) string {
	if row.Status == task.StatusInProgress && row.ClaimedBy != "" {
		return row.Status + " · " + row.ClaimedBy
	}
	return row.Status
}

// humanDrift owns the human words for every drift class in one place.
func humanDrift(row app.Row) string {
	switch row.Drift {
	case app.DriftStatus:
		return "⚠ board says " + row.RowStatus
	case app.DriftNoRow:
		return "⚠ board says missing"
	case app.DriftNoFile:
		return "⚠ no file"
	case app.DriftBadFile:
		return "⚠ unparsable"
	}
	return ""
}

// tsvLine's column order is a wire contract: new fields append (updated, then claimed-by, waits).
func tsvLine(row app.Row) string {
	return tsv(row.ID, row.Board, row.Status, row.Title, agentDrift(row),
		row.UpdatedAt.Format(time.RFC3339), row.ClaimedBy, strings.Join(row.Waits, ","))
}

// agentDrift owns the TSV token for every drift class in one place.
func agentDrift(row app.Row) string {
	switch row.Drift {
	case app.DriftStatus:
		return "board=" + row.RowStatus
	case app.DriftNoRow:
		return "no-row"
	case app.DriftNoFile:
		return "no-file"
	case app.DriftBadFile:
		return "bad-file"
	}
	return ""
}

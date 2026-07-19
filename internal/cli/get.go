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

// taskKeyWidth must match the widest key rendered by kv (currently "blocked-by:").
const taskKeyWidth = len("blocked-by:")

func newGetCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	cmd := newGroupCmd("get", "read tasks and tracks from the files", getUsage, getExample)
	cmd.AddCommand(
		newGetTaskCmd(a, cwd, stdout),
		newGetTasksCmd(a, cwd, stdout, "tasks", false),
		newGetTracksCmd(a, cwd, stdout),
		newGetNextCmd(a, cwd, stdout),
	)
	return cmd
}

func newListCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
	return newGetTasksCmd(a, cwd, stdout, "list", true)
}

func newGetTaskCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
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
			return getTaskOut(a, cwd, args[0], section, body, agent, stdout)
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, track, status, title, gates-done, gates-total, blocked-by, path, updated, claimed-by")
	cmd.Flags().StringVar(&section, "section", "", "print only this section's body")
	cmd.Flags().BoolVar(&body, "body", false, "print the whole body below the frontmatter")
	cmd.MarkFlagsMutuallyExclusive("body", "section")
	cmd.MarkFlagsMutuallyExclusive("body", "agent")
	return cmd
}

func getTaskOut(a app.App, cwd, id, section string, body, agent bool, stdout io.Writer) error {
	if body {
		text, err := a.Body(cwd, id)
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, text)
		return nil
	}
	if section != "" {
		sec, err := a.Section(cwd, id, section)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, sec)
		return nil
	}
	info, err := a.GetTask(cwd, id)
	if err != nil {
		return err
	}
	if agent {
		fmt.Fprintln(stdout, taskAgent(info))
		return nil
	}
	fmt.Fprintln(stdout, taskHuman(info))
	return nil
}

func newGetTracksCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
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
			tracks, err := a.GetTracks(cwd, in)
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
	addInFlag(cmd, &in)
	return cmd
}

// newGetNextCmd's --claim atomically marks the task in-progress so parallel
// agents each land on a distinct task.
func newGetNextCmd(a app.App, cwd string, stdout io.Writer) *cobra.Command {
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
			info, err := a.GetNext(cwd, in, claim, claimer(as))
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

func newGetTasksCmd(a app.App, cwd string, stdout io.Writer, use string, hidden bool) *cobra.Command {
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
			rows, err := a.List(cwd, status, in)
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
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: id, board-path, status, title, drift, updated")
	addInFlag(cmd, &in)
	return cmd
}

func taskHuman(info app.TaskInfo) string {
	var b strings.Builder
	kv(&b, "id", info.ID)
	kv(&b, "title", info.Title)
	kv(&b, "status", info.Status)
	if info.ClaimedBy != "" {
		kv(&b, "claimed-by", info.ClaimedBy)
	}
	kv(&b, "track", info.Track)
	kv(&b, "blocked-by", blockedByHuman(info.Deps))
	kv(&b, "gates", fmt.Sprintf("%d/%d", info.GatesDone, info.GatesTotal))
	kv(&b, "updated", humanize.RelTime(info.UpdatedAt, time.Now()))
	kv(&b, "path", info.Path)
	return strings.TrimRight(b.String(), "\n")
}

func kv(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%-*s %s\n", taskKeyWidth, key+":", value)
}

func blockedByHuman(deps []app.Dep) string {
	if len(deps) == 0 {
		return "none"
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = d.ID + " (" + d.Status + ")"
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
		counts := fmt.Sprintf("%d todo · %d in-progress · %d done · %d blocked · %d archived",
			tr.Todo, tr.InProgress, tr.Done, tr.Blocked, tr.Archived)
		lines = append(lines, fmt.Sprintf("%-*s  %-*s  %s", pathW, tr.Path, titleW, tr.Title, counts))
	}
	return lines
}

// trackAgent's column order is a wire contract; don't reorder them.
func trackAgent(tr app.TrackInfo) string {
	return tsv(
		tr.Path,
		tr.Title,
		strconv.Itoa(tr.Todo),
		strconv.Itoa(tr.InProgress),
		strconv.Itoa(tr.Done),
		strconv.Itoa(tr.Blocked),
		strconv.Itoa(tr.Archived),
	)
}

// humanLine renders "[track/ ]id  status  title[  drift]  age[  waits]".
func humanLine(r app.Row) string {
	var b strings.Builder
	if r.Board != "." {
		b.WriteString(r.Board)
		b.WriteString("/ ")
	}
	b.WriteString(r.ID)
	b.WriteString("  ")
	b.WriteString(statusCell(r))
	b.WriteString("  ")
	b.WriteString(r.Title)
	if drift := humanDrift(r); drift != "" {
		b.WriteString("  ")
		b.WriteString(drift)
	}
	b.WriteString("  ")
	b.WriteString(ageStyle.Render(humanize.RelTime(r.UpdatedAt, time.Now())))
	if wait := waitsCell(r); wait != "" {
		b.WriteString("  ")
		b.WriteString(ageStyle.Render(wait))
	}
	return b.String()
}

func waitsCell(r app.Row) string {
	if len(r.Waits) == 0 {
		return ""
	}
	return "waits " + strings.Join(r.Waits, ",")
}

func statusCell(r app.Row) string {
	if r.Status == task.StatusInProgress && r.ClaimedBy != "" {
		return r.Status + " · " + r.ClaimedBy
	}
	return r.Status
}

func humanDrift(r app.Row) string {
	switch {
	case r.RowMissing:
		return "⚠ board says missing"
	case r.RowStatus != "":
		return "⚠ board says " + r.RowStatus
	}
	return ""
}

// tsvLine's column order is a wire contract; updated was appended last so
// existing parsers keep working.
func tsvLine(r app.Row) string {
	return tsv(r.ID, r.Board, r.Status, r.Title, agentDrift(r), r.UpdatedAt.Format(time.RFC3339))
}

func agentDrift(r app.Row) string {
	switch {
	case r.RowMissing:
		return "no-row"
	case r.RowStatus != "":
		return "board=" + r.RowStatus
	}
	return ""
}

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/fsys"
	"github.com/raphaelCamblong/duty/internal/tree"
	"github.com/raphaelCamblong/duty/internal/watch"
)

const (
	watchUsage   = "usage: duty watch [--agent] [--in PATH]"
	watchExample = `  duty watch --agent`
)

// newWatchCmd builds watch: a long-running stream of one line per task state
// change, powered by the same filesystem watcher the TUI uses.
func newWatchCmd(a app.App, f fsys.FS, cwd string, stdout io.Writer) *cobra.Command {
	var (
		agent bool
		in    string
	)
	cmd := &cobra.Command{
		Use:     "watch",
		Short:   "stream one line per task state change (long-running)",
		Example: watchExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New(watchUsage)
			}
			return runWatch(a, f, cwd, in, agent, stdout)
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: time, event, id, field, old, new")
	addInFlag(cmd, &in)
	return cmd
}

// runWatch streams task-state changes until interrupted: on each debounced
// filesystem burst it re-snapshots, diffs against the previous snapshot, and
// prints one line per change. It prints nothing on start (state, not history),
// returns nil on SIGINT, and returns an error only when the tree becomes
// unreadable — the tree disappearing mid-watch.
func runWatch(a app.App, f fsys.FS, cwd, in string, agent bool, stdout io.Writer) error {
	root, err := tree.FindRoot(f, cwd)
	if err != nil {
		return err
	}
	prev, err := a.Snapshot(cwd, in)
	if err != nil {
		return err
	}
	w, err := watch.NewWatcher(f, root)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-w.C:
			if !ok {
				return nil
			}
			cur, err := a.Snapshot(cwd, in)
			if err != nil {
				return err
			}
			emit(stdout, app.Diff(prev, cur), agent)
			prev = cur
		}
	}
}

// emit writes one line per event: a readable line, or the TSV record when agent
// is set. Every line in a burst shares the emit timestamp.
func emit(w io.Writer, events []app.Event, agent bool) {
	now := time.Now()
	for _, e := range events {
		if agent {
			fmt.Fprintln(w, watchTSV(now, e))
			continue
		}
		fmt.Fprintln(w, watchHuman(now, e))
	}
}

// watchTSV renders one change as the agent record:
// time<TAB>event<TAB>id<TAB>field<TAB>old<TAB>new, time in RFC3339.
func watchTSV(now time.Time, e app.Event) string {
	return strings.Join([]string{now.Format(time.RFC3339), e.Kind, e.ID, e.Field, e.Old, e.New}, "\t")
}

// watchHuman renders one change as a readable line: a short clock time, the
// task id, the event, and the old → new transition.
func watchHuman(now time.Time, e app.Event) string {
	return now.Format("15:04:05") + "  " + humanChange(e)
}

// humanChange renders an event's body without its timestamp.
func humanChange(e app.Event) string {
	switch e.Kind {
	case app.EventCreated:
		return fmt.Sprintf("%s created · %s", e.ID, e.New)
	case app.EventDeleted:
		return fmt.Sprintf("%s deleted", e.ID)
	default:
		return fmt.Sprintf("%s %s %s → %s", e.ID, e.Kind, orDash(e.Old), orDash(e.New))
	}
}

// orDash renders an empty value as an em dash so a transition reads cleanly.
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

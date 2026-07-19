package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
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

// watchCmd is the watch command's dependencies and output sink.
type watchCmd struct {
	app app.App
	fs  fsys.FS
	cwd string
	out io.Writer
}

func newWatchCmd(svc app.App, fs fsys.FS, cwd string, stdout io.Writer) *cobra.Command {
	wc := watchCmd{app: svc, fs: fs, cwd: cwd, out: stdout}
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
			return runWatch(wc, in, agent)
		},
	}
	cmd.Flags().BoolVar(&agent, "agent", false, "TSV output: time, event, id, field, old, new")
	addInFlag(cmd, &in)
	return cmd
}

// runWatch prints nothing for the initial snapshot, only for later changes;
// it returns nil on SIGINT and an error only if the tree becomes unreadable.
func runWatch(wc watchCmd, in string, agent bool) error {
	root, err := tree.FindRoot(wc.fs, wc.cwd)
	if err != nil {
		return err
	}
	prev, err := wc.app.Snapshot(app.Scope{Cwd: wc.cwd, In: in})
	if err != nil {
		return err
	}
	watcher, err := watch.NewWatcher(wc.fs, root)
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-watcher.C:
			if !ok {
				return nil
			}
			next, err := wc.emitChanges(prev, in, agent)
			if err != nil {
				return err
			}
			prev = next
		}
	}
}

// emitChanges re-snapshots the tree, prints what changed against prev, and
// returns the new baseline.
func (wc watchCmd) emitChanges(prev map[string]app.TaskState, in string, agent bool) (map[string]app.TaskState, error) {
	cur, err := wc.app.Snapshot(app.Scope{Cwd: wc.cwd, In: in})
	if err != nil {
		return nil, err
	}
	emit(wc.out, app.Diff(prev, cur), agent)
	return cur, nil
}

// emit gives every event in a burst the same timestamp (capture time, not per-event).
func emit(out io.Writer, events []app.Event, agent bool) {
	now := time.Now()
	for _, event := range events {
		if agent {
			fmt.Fprintln(out, watchTSV(now, event))
			continue
		}
		fmt.Fprintln(out, watchHuman(now, event))
	}
}

// watchTSV's column order (time, event, id, field, old, new) is a wire contract.
func watchTSV(now time.Time, event app.Event) string {
	return tsv(now.Format(time.RFC3339), event.Kind, event.ID, event.Field, event.Old, event.New)
}

func watchHuman(now time.Time, event app.Event) string {
	return now.Format("15:04:05") + "  " + humanChange(event)
}

func humanChange(event app.Event) string {
	switch event.Kind {
	case app.EventCreated:
		return fmt.Sprintf("%s created · %s", event.ID, event.New)
	case app.EventDeleted:
		return fmt.Sprintf("%s deleted", event.ID)
	default:
		return fmt.Sprintf("%s %s %s → %s", event.ID, event.Kind, orDash(event.Old), orDash(event.New))
	}
}

func orDash(text string) string {
	if text == "" {
		return "—"
	}
	return text
}

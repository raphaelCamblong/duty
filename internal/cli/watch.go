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

func newWatchCmd(a app.App, f fsys.FS, cwd string, stdout io.Writer) *cobra.Command {
	wc := watchCmd{app: a, fs: f, cwd: cwd, out: stdout}
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
	prev, err := wc.app.Snapshot(wc.cwd, in)
	if err != nil {
		return err
	}
	w, err := watch.NewWatcher(wc.fs, root)
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
			cur, err := wc.app.Snapshot(wc.cwd, in)
			if err != nil {
				return err
			}
			emit(wc.out, app.Diff(prev, cur), agent)
			prev = cur
		}
	}
}

// emit gives every event in a burst the same timestamp (capture time, not per-event).
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

// watchTSV's column order (time, event, id, field, old, new) is a wire contract.
func watchTSV(now time.Time, e app.Event) string {
	return tsv(now.Format(time.RFC3339), e.Kind, e.ID, e.Field, e.Old, e.New)
}

func watchHuman(now time.Time, e app.Event) string {
	return now.Format("15:04:05") + "  " + humanChange(e)
}

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

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

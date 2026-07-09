package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphaelCamblong/duty/internal/board"
	"github.com/raphaelCamblong/duty/internal/task"
	"github.com/raphaelCamblong/duty/internal/tree"
)

const listUsage = "usage: duty list [--status S] [--agent]"

// listRow is one open task as read from its file, plus the drift computed
// against its board row.
type listRow struct {
	id, title, status      string
	prefix, boardPath      string // prefix: human "sub/board/"; boardPath: "." or "sub/board"
	driftHuman, driftAgent string
}

// runList prints every open task in the current board and every board below
// it, read from the files (never the board index). --status filters by
// status; --agent switches to stable TSV for scripts.
func runList(cwd string, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("status", "", "list only this status")
	agent := fs.Bool("agent", false, "TSV output: id, board-path, status, title, drift")
	pos, err := positionals(fs, args, listUsage)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return errors.New(listUsage)
	}
	if *status != "" && !task.ValidStatus(*status) {
		return unknownStatusErr(*status)
	}

	boardDir, err := tree.CurrentBoard(cwd)
	if err != nil {
		return err
	}
	boards, err := tree.Boards(boardDir)
	if err != nil {
		return err
	}
	for _, b := range boards {
		rows, err := listBoardRows(boardDir, b)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if *status != "" && r.status != *status {
				continue
			}
			if *agent {
				fmt.Fprintln(stdout, r.tsv())
				continue
			}
			fmt.Fprintln(stdout, r.line())
		}
	}
	return nil
}

// listBoardRows returns one listRow per task file directly in board b (its
// sub-boards are separate entries in the caller's board list), tagged with
// its path relative to root — the board list started from.
func listBoardRows(root, b string) ([]listRow, error) {
	rel, err := filepath.Rel(root, b)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", b, err)
	}
	boardPath := "."
	prefix := ""
	if rel != "." {
		boardPath = filepath.ToSlash(rel)
		prefix = boardPath + "/"
	}

	indexPath := filepath.Join(b, boardFile)
	index, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(b)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", b, err)
	}

	var rows []listRow
	for _, e := range entries {
		if e.IsDir() || !tree.IsTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(b, e.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		t, err := task.Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		row, ok := board.FindRow(index, e.Name())
		human, agent := drift(ok, row, t.Status)
		rows = append(rows, listRow{
			id: t.ID, title: t.Title, status: t.Status,
			prefix: prefix, boardPath: boardPath,
			driftHuman: human, driftAgent: agent,
		})
	}
	return rows, nil
}

// drift compares a task's file status to its board row, found via
// board.FindRow: rowOK false means the row is missing entirely. It returns
// the flag rendered for a human ("" when in sync) and for --agent ("",
// "board=<status>", or "no-row").
func drift(rowOK bool, row, fileStatus string) (human, agent string) {
	if !rowOK {
		return "⚠ board says missing", "no-row"
	}
	rowStatus, ok := board.RowStatus(row)
	if !ok || rowStatus == fileStatus {
		return "", ""
	}
	return "⚠ board says " + rowStatus, "board=" + rowStatus
}

// line renders r for human reading: "[prefix ]id  status  title[  drift]".
func (r listRow) line() string {
	var b strings.Builder
	if r.prefix != "" {
		b.WriteString(r.prefix)
		b.WriteByte(' ')
	}
	b.WriteString(r.id)
	b.WriteString("  ")
	b.WriteString(r.status)
	b.WriteString("  ")
	b.WriteString(r.title)
	if r.driftHuman != "" {
		b.WriteString("  ")
		b.WriteString(r.driftHuman)
	}
	return b.String()
}

// tsv renders r as one agent-output record:
// id<TAB>board-path<TAB>status<TAB>title<TAB>drift.
func (r listRow) tsv() string {
	return strings.Join([]string{r.id, r.boardPath, r.status, r.title, r.driftAgent}, "\t")
}

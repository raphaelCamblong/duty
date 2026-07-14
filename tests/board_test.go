package tests

import (
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/board"
)

// boardFixture is a board with hand-written prose, a banner, and unusual row
// spacing — everything the surgical-edit invariant must preserve. Tests build
// their expected outputs from these same lines, so any byte drift in an
// untouched line fails the comparison.
func boardFixture() []string {
	return []string{
		"# My Project — ops board", // 0
		"",
		"> hand-written banner: keep me byte-identical", // 2
		"",
		"Convention: [README.md](README.md). Workers update their row's status via the CLI.",
		"Order top-to-bottom is the intended build order.", // 5
		"",
		"Some hand-written prose the tooling must never touch.", // 7
		"",
		"## Boards", // 9
		"",
		"- [backend/](backend/BOARD.md) — Backend", // 11
		"",
		"## Open tasks", // 13
		"",
		"| Task | Title | Status |", // 15
		"|------|-------|--------|",
		"| [T-01](T-01-first-task.md) | First task | todo |", // 17
		"|  [T-02](T-02-weird-spacing.md)   |   Weird   spacing kept   |  in-progress  |",
		"| [T-03](T-03-third-task.md) | Third task | blocked |", // 19
		"",
		"## Later", // 21
		"",
		"| Task | Title | Status |", // 23
		"|------|-------|--------|",
		"| [T-04](T-04-someday.md) | Someday | todo |", // 25
		"",
		"Completed tasks (7) archived: [archive/](archive/).", // 27
		"",
	}
}

func joinBoard(lines []string) []byte {
	return []byte(strings.Join(lines, "\n"))
}

// replaceLine returns a copy of lines with line i swapped for repl.
func replaceLine(lines []string, i int, repl string) []string {
	out := append([]string(nil), lines...)
	out[i] = repl
	return out
}

// insertLines returns a copy of lines with add inserted before index i.
func insertLines(lines []string, i int, add ...string) []string {
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:i]...)
	out = append(out, add...)
	out = append(out, lines[i:]...)
	return out
}

// removeLines returns a copy of lines without the half-open range [i, j).
func removeLines(lines []string, i, j int) []string {
	out := make([]string, 0, len(lines)-(j-i))
	out = append(out, lines[:i]...)
	out = append(out, lines[j:]...)
	return out
}

func TestRender(t *testing.T) {
	want := strings.Join([]string{
		"# Board",
		"",
		"Convention: [README.md](README.md). Workers update their row's status via the CLI.",
		"Order top-to-bottom is the intended build order.",
		"",
		"## Open tasks",
		"",
		"| Task | Title | Status |",
		"|------|-------|--------|",
		"",
		"Completed tasks (0) archived: [archive/](archive/).",
		"",
	}, "\n")

	t.Run("skeleton matches the spec shape", func(t *testing.T) {
		if got := string(board.Render("Board")); got != want {
			t.Errorf("Render() =\n%s\nwant:\n%s", got, want)
		}
	})
	t.Run("title round-trips through Title", func(t *testing.T) {
		if got := board.Title(board.Render("Backend")); got != "Backend" {
			t.Errorf("Title(Render(%q)) = %q, want %q", "Backend", got, "Backend")
		}
	})
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"fixture H1", strings.Join(boardFixture(), "\n"), "My Project — ops board"},
		{"H1 with trailing spaces", "# Spaced   \n\ntext\n", "Spaced"},
		{"no H1 at all", "just prose\n\n## Open tasks\n", ""},
		{"section heading is not an H1", "## Open tasks\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := board.Title([]byte(tt.content)); got != tt.want {
				t.Errorf("Title() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindRow(t *testing.T) {
	fixture := boardFixture()
	prosey := insertLines(fixture, 8, "See (T-77-ghost.md) for background.")
	tests := []struct {
		name     string
		content  []byte
		filename string
		wantRow  string
		wantOK   bool
	}{
		{
			name:     "finds a row byte-identical",
			content:  joinBoard(fixture),
			filename: "T-02-weird-spacing.md",
			wantRow:  fixture[18],
			wantOK:   true,
		},
		{
			name:     "finds a row in a later section",
			content:  joinBoard(fixture),
			filename: "T-04-someday.md",
			wantRow:  fixture[25],
			wantOK:   true,
		},
		{
			name:     "missing filename",
			content:  joinBoard(fixture),
			filename: "T-99-nope.md",
			wantOK:   false,
		},
		{
			name:     "filename in prose is not a row",
			content:  joinBoard(prosey),
			filename: "T-77-ghost.md",
			wantOK:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, ok := board.FindRow(tt.content, tt.filename)
			if ok != tt.wantOK {
				t.Fatalf("FindRow() ok = %v, want %v", ok, tt.wantOK)
			}
			if row != tt.wantRow {
				t.Errorf("FindRow() row = %q, want %q", row, tt.wantRow)
			}
		})
	}
}

func TestRowStatus(t *testing.T) {
	fixture := boardFixture()
	tests := []struct {
		name       string
		row        string
		wantStatus string
		wantOK     bool
	}{
		{name: "plain row", row: fixture[17], wantStatus: "todo", wantOK: true},
		{name: "unusual spacing kept in the row still yields a trimmed status", row: fixture[18], wantStatus: "in-progress", wantOK: true},
		{name: "not a table row", row: "not a row at all", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := board.RowStatus(tt.row)
			if ok != tt.wantOK {
				t.Fatalf("RowStatus() ok = %v, want %v", ok, tt.wantOK)
			}
			if status != tt.wantStatus {
				t.Errorf("RowStatus() = %q, want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestAddRow(t *testing.T) {
	fixture := boardFixture()
	newRow := "| [T-05](T-05-new-task.md) | New task | todo |"
	tests := []struct {
		name    string
		content []string
		section string
		want    []string
		wantErr bool
	}{
		{
			name:    "appends to the default section table",
			content: fixture,
			section: "Open tasks",
			want:    insertLines(fixture, 20, newRow),
		},
		{
			name:    "appends to a later section table",
			content: fixture,
			section: "Later",
			want:    insertLines(fixture, 26, newRow),
		},
		{
			name:    "creates a missing section above the footer",
			content: fixture,
			section: "Blocked on infra",
			want: insertLines(
				fixture, 27,
				"## Blocked on infra",
				"",
				"| Task | Title | Status |",
				"|------|-------|--------|",
				newRow,
				"",
			),
		},
		{
			name:    "errors when creating a section on a footerless board",
			content: removeLines(fixture, 27, 28),
			section: "Blocked on infra",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.AddRow(joinBoard(tt.content), tt.section, "T-05", "T-05-new-task.md", "New task", "todo")
			if tt.wantErr {
				if err == nil {
					t.Fatal("AddRow() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("AddRow() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("AddRow() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSetRowStatus(t *testing.T) {
	fixture := boardFixture()
	tests := []struct {
		name     string
		filename string
		want     []string
		wantErr  bool
	}{
		{
			name:     "rewrites only the status cell",
			filename: "T-01-first-task.md",
			want:     replaceLine(fixture, 17, "| [T-01](T-01-first-task.md) | First task | done |"),
		},
		{
			name:     "keeps unusual spacing in the untouched cells",
			filename: "T-02-weird-spacing.md",
			want:     replaceLine(fixture, 18, "|  [T-02](T-02-weird-spacing.md)   |   Weird   spacing kept   | done |"),
		},
		{
			name:     "errors on a missing row",
			filename: "T-99-nope.md",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.SetRowStatus(joinBoard(fixture), tt.filename, "done")
			if tt.wantErr {
				if err == nil {
					t.Fatal("SetRowStatus() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("SetRowStatus() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("SetRowStatus() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestDropRow(t *testing.T) {
	fixture := boardFixture()
	tests := []struct {
		name     string
		filename string
		want     []string
		wantErr  bool
	}{
		{
			name:     "removes exactly the row line",
			filename: "T-02-weird-spacing.md",
			want:     removeLines(fixture, 18, 19),
		},
		{
			name:     "removes the last row of a section",
			filename: "T-04-someday.md",
			want:     removeLines(fixture, 25, 26),
		},
		{
			name:     "errors on a missing row",
			filename: "T-99-nope.md",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.DropRow(joinBoard(fixture), tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DropRow() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DropRow() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("DropRow() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestMoveRow(t *testing.T) {
	fixture := boardFixture()
	weirdRow := fixture[18]
	tests := []struct {
		name     string
		filename string
		section  string
		want     []string
		wantErr  bool
	}{
		{
			name:     "moves the row byte-identical into an existing section",
			filename: "T-02-weird-spacing.md",
			section:  "Later",
			// Drop line 18, then append after the T-04 row (now at 24).
			want: insertLines(removeLines(fixture, 18, 19), 25, weirdRow),
		},
		{
			name:     "creates the target section above the footer",
			filename: "T-04-someday.md",
			section:  "Doing",
			// Drop line 25; the footer is now at 26.
			want: insertLines(
				removeLines(fixture, 25, 26), 26,
				"## Doing",
				"",
				"| Task | Title | Status |",
				"|------|-------|--------|",
				fixture[25],
				"",
			),
		},
		{
			name:     "errors on a missing row",
			filename: "T-99-nope.md",
			section:  "Later",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.MoveRow(joinBoard(fixture), tt.filename, tt.section)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MoveRow() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("MoveRow() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("MoveRow() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestPruneEmptySections(t *testing.T) {
	fixture := boardFixture()
	// "Later" emptied: T-04's row dropped, scaffolding left behind.
	emptied := removeLines(fixture, 25, 26)
	proseSection := []string{
		"# Board",
		"",
		"## Open tasks",
		"",
		"| Task | Title | Status |",
		"|------|-------|--------|",
		"",
		"## Notes",
		"",
		"Hand-written notes, no table at all.",
		"",
		"Completed tasks (0) archived: [archive/](archive/).",
		"",
	}
	twoEmpty := []string{
		"# Board",
		"",
		"## Open tasks",
		"",
		"| Task | Title | Status |",
		"|------|-------|--------|",
		"| [T-01](T-01-a.md) | A | todo |",
		"",
		"## Doing", // 8
		"",
		"| Task | Title | Status |",
		"|------|-------|--------|",
		"",
		"## Done here", // 13
		"",
		"| Task | Title | Status |",
		"|------|-------|--------|",
		"",
		"Completed tasks (2) archived: [archive/](archive/).",
		"",
	}
	tests := []struct {
		name    string
		content []string
		want    []string
	}{
		{
			name:    "removes a section whose table lost its last row",
			content: emptied,
			want:    removeLines(emptied, 21, 26),
		},
		{
			name:    "never removes the default section",
			content: strings.Split(string(board.Render("Board")), "\n"),
			want:    strings.Split(string(board.Render("Board")), "\n"),
		},
		{
			name:    "keeps sections holding rows, bullets, or prose",
			content: fixture,
			want:    fixture,
		},
		{
			name:    "keeps a section holding only prose",
			content: proseSection,
			want:    proseSection,
		},
		{
			name:    "removes every empty section in one pass",
			content: twoEmpty,
			want:    removeLines(removeLines(twoEmpty, 13, 18), 8, 13),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := board.PruneEmptySections(joinBoard(tt.content))
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("PruneEmptySections() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSetArchivedCount(t *testing.T) {
	fixture := boardFixture()
	tests := []struct {
		name    string
		content []string
		n       int
		want    []string
		wantErr bool
	}{
		{
			name:    "rewrites only the footer count",
			content: fixture,
			n:       12,
			want:    replaceLine(fixture, 27, "Completed tasks (12) archived: [archive/](archive/)."),
		},
		{
			name:    "rewrites back down to zero",
			content: fixture,
			n:       0,
			want:    replaceLine(fixture, 27, "Completed tasks (0) archived: [archive/](archive/)."),
		},
		{
			name:    "errors when the footer is missing",
			content: removeLines(fixture, 27, 28),
			n:       3,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.SetArchivedCount(joinBoard(tt.content), tt.n)
			if tt.wantErr {
				if err == nil {
					t.Fatal("SetArchivedCount() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("SetArchivedCount() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("SetArchivedCount() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestAddBoardBullet(t *testing.T) {
	fixture := boardFixture()
	skeleton := strings.Split(string(board.Render("Board")), "\n")
	tests := []struct {
		name    string
		content []string
		want    []string
	}{
		{
			name:    "appends after the last existing bullet",
			content: fixture,
			want:    insertLines(fixture, 12, "- [frontend/](frontend/BOARD.md) — Frontend"),
		},
		{
			name:    "creates the Boards section before the first task section",
			content: skeleton,
			want: insertLines(
				skeleton, 5,
				"## Boards",
				"",
				"- [frontend/](frontend/BOARD.md) — Frontend",
				"",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := board.AddBoardBullet(joinBoard(tt.content), "frontend", "Frontend")
			if err != nil {
				t.Fatalf("AddBoardBullet() error = %v", err)
			}
			if want := joinBoard(tt.want); string(got) != string(want) {
				t.Errorf("AddBoardBullet() =\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestSurgicalRoundTrip chains add → status → move (creating a section) →
// move back → prune → drop → footer bump and back. Ending byte-identical to
// the starting fixture proves every operation preserves everything it does
// not own — the board-side half of the spec §6 round-trip invariant.
func TestSurgicalRoundTrip(t *testing.T) {
	start := joinBoard(boardFixture())

	b, err := board.AddRow(start, "Open tasks", "T-99", "T-99-scratch.md", "Scratch", "todo")
	if err != nil {
		t.Fatalf("AddRow() error = %v", err)
	}
	if b, err = board.SetRowStatus(b, "T-99-scratch.md", "in-progress"); err != nil {
		t.Fatalf("SetRowStatus() error = %v", err)
	}
	if b, err = board.MoveRow(b, "T-99-scratch.md", "Temp"); err != nil {
		t.Fatalf("MoveRow() to new section error = %v", err)
	}
	if b, err = board.MoveRow(b, "T-99-scratch.md", "Open tasks"); err != nil {
		t.Fatalf("MoveRow() back error = %v", err)
	}
	b = board.PruneEmptySections(b)
	if b, err = board.DropRow(b, "T-99-scratch.md"); err != nil {
		t.Fatalf("DropRow() error = %v", err)
	}
	if b, err = board.SetArchivedCount(b, 8); err != nil {
		t.Fatalf("SetArchivedCount() error = %v", err)
	}
	if b, err = board.SetArchivedCount(b, 7); err != nil {
		t.Fatalf("SetArchivedCount() error = %v", err)
	}

	if string(b) != string(start) {
		t.Errorf("round trip diverged:\ngot:\n%s\nwant:\n%s", b, start)
	}
}

func TestSections(t *testing.T) {
	got := board.Sections(joinBoard(boardFixture()))
	want := []board.Section{
		{Name: "Open tasks", Rows: []board.Row{
			{ID: "T-01", File: "T-01-first-task.md", Title: "First task", Status: "todo"},
			{ID: "T-02", File: "T-02-weird-spacing.md", Title: "Weird   spacing kept", Status: "in-progress"},
			{ID: "T-03", File: "T-03-third-task.md", Title: "Third task", Status: "blocked"},
		}},
		{Name: "Later", Rows: []board.Row{
			{ID: "T-04", File: "T-04-someday.md", Title: "Someday", Status: "todo"},
		}},
	}
	if len(got) != len(want) {
		t.Fatalf("Sections() = %d sections, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name {
			t.Errorf("section %d name = %q, want %q", i, got[i].Name, want[i].Name)
		}
		if len(got[i].Rows) != len(want[i].Rows) {
			t.Fatalf("section %q = %d rows, want %d", want[i].Name, len(got[i].Rows), len(want[i].Rows))
		}
		for j, r := range want[i].Rows {
			if got[i].Rows[j] != r {
				t.Errorf("section %q row %d = %+v, want %+v", want[i].Name, j, got[i].Rows[j], r)
			}
		}
	}
}

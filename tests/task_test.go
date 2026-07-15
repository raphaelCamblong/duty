package tests

import (
	"slices"
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/task"
)

func TestRenderParseRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		title     string
		blockedBy []string
	}{
		{
			name:  "no blockers",
			id:    "T-01",
			title: "Short imperative title",
		},
		{
			name:      "one blocker",
			id:        "T-02",
			title:     "Task file domain package",
			blockedBy: []string{"T-01"},
		},
		{
			name:      "several blockers",
			id:        "T-06",
			title:     "CLI dispatch, init, create, board",
			blockedBy: []string{"T-02", "T-03", "T-04"},
		},
		{
			name:  "title with colon-space needs quoting",
			id:    "T-12",
			title: "duty: wire the watcher",
		},
		{
			name:  "title with space-hash needs quoting",
			id:    "T-13",
			title: "fix #42 for real",
		},
		{
			name:  "title with double quote",
			id:    "T-14",
			title: `rename the "board" concept`,
		},
		{
			name:  "title with apostrophe stays plain",
			id:    "T-15",
			title: "don't re-decide scope",
		},
		{
			name:  "unicode title",
			id:    "T-16",
			title: "éliminer les tâches mortes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := task.Render(tt.id, tt.title, tt.blockedBy)
			got, err := task.Parse(rendered)
			if err != nil {
				t.Fatalf("Parse(Render()) error = %v\nrendered:\n%s", err, rendered)
			}
			if got.ID != tt.id {
				t.Errorf("ID = %q, want %q", got.ID, tt.id)
			}
			if got.Title != tt.title {
				t.Errorf("Title = %q, want %q", got.Title, tt.title)
			}
			if got.Status != task.StatusTodo {
				t.Errorf("Status = %q, want %q", got.Status, task.StatusTodo)
			}
			if !slices.Equal(got.BlockedBy, tt.blockedBy) {
				t.Errorf("BlockedBy = %v, want %v", got.BlockedBy, tt.blockedBy)
			}
		})
	}
}

func TestRenderSkeleton(t *testing.T) {
	rendered := string(task.Render("T-09", "Spec invariants test suite", []string{"T-08"}))

	want := `---
id: T-09
title: Spec invariants test suite
status: todo
blocked-by: [T-08]
---

# T-09 — Spec invariants test suite

## Goal

## Read first

## Scope

## Out of scope

## Gates

## Report
`
	if rendered != want {
		t.Errorf("Render() =\n%s\nwant:\n%s", rendered, want)
	}

	done, total := task.CountGates([]byte(rendered))
	if done != 0 || total != 0 {
		t.Errorf("CountGates(fresh render) = %d/%d, want 0/0", done, total)
	}
}

func TestRenderWithBody(t *testing.T) {
	const body = "## Goal\nShip it.\n\n## Gates\n- [ ] build\n- [x] tests\n\n## Report\n"
	want := `---
id: T-20
title: One-shot
status: todo
blocked-by: [T-01]
---

# T-20 — One-shot

` + body
	t.Run("splices the body verbatim below the generated H1", func(t *testing.T) {
		if got := string(task.RenderWithBody("T-20", "One-shot", []string{"T-01"}, []byte(body))); got != want {
			t.Errorf("RenderWithBody =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("trims leading blank lines and guarantees a trailing newline", func(t *testing.T) {
		got := string(task.RenderWithBody("T-20", "One-shot", []string{"T-01"}, []byte("\n\n"+strings.TrimRight(body, "\n"))))
		if got != want {
			t.Errorf("RenderWithBody(padded) =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("counts gates authored in the body", func(t *testing.T) {
		done, total := task.CountGates(task.RenderWithBody("T-20", "One-shot", nil, []byte(body)))
		if done != 1 || total != 2 {
			t.Errorf("CountGates(--body) = %d/%d, want 1/2", done, total)
		}
	})
}

func TestOpensAtSection(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{name: "flush heading", content: "## Goal\nx\n", want: true},
		{name: "leading blank lines", content: "\n\n## Goal\nx\n", want: true},
		{name: "prose before the heading", content: "intro\n## Goal\n", want: false},
		{name: "no heading at all", content: "just prose\n", want: false},
		{name: "empty", content: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := task.OpensAtSection([]byte(tt.content)); got != tt.want {
				t.Errorf("OpensAtSection(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    task.Task
		wantErr bool
	}{
		{
			name: "spec example with inline comments",
			content: "---\n" +
				"id: T-01\n" +
				"title: Short imperative title\n" +
				"status: todo            # todo | in-progress | done | blocked\n" +
				"blocked-by: []          # task ids that must be done first\n" +
				"---\n\n# T-01 — Short imperative title\n",
			want: task.Task{ID: "T-01", Title: "Short imperative title", Status: "todo"},
		},
		{
			name: "blocked-by list",
			content: "---\nid: T-07\ntitle: CLI status\nstatus: in-progress\n" +
				"blocked-by: [T-02, T-03]\n---\nbody\n",
			want: task.Task{
				ID: "T-07", Title: "CLI status", Status: "in-progress",
				BlockedBy: []string{"T-02", "T-03"},
			},
		},
		{
			name:    "missing frontmatter",
			content: "# Just a heading\n\nNo frontmatter here.\n",
			wantErr: true,
		},
		{
			name:    "unterminated frontmatter",
			content: "---\nid: T-01\ntitle: X\nstatus: todo\n",
			wantErr: true,
		},
		{
			name:    "frontmatter not at byte zero",
			content: "\n---\nid: T-01\ntitle: X\nstatus: todo\nblocked-by: []\n---\n",
			wantErr: true,
		},
		{
			name:    "invalid yaml in frontmatter",
			content: "---\nid: [unclosed\n---\nbody\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := task.Parse([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() = %+v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.ID != tt.want.ID || got.Title != tt.want.Title || got.Status != tt.want.Status {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
			if !slices.Equal(got.BlockedBy, tt.want.BlockedBy) {
				t.Errorf("BlockedBy = %v, want %v", got.BlockedBy, tt.want.BlockedBy)
			}
		})
	}
}

func TestSetStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		status  string
		want    string
		wantErr bool
	}{
		{
			name: "changes exactly the status line, decoy in body survives",
			content: "---\nid: T-05\ntitle: X\nstatus: todo\nblocked-by: []\n---\n\n" +
				"# T-05 — X\n\nstatus: decoy stays untouched\n",
			status: "in-progress",
			want: "---\nid: T-05\ntitle: X\nstatus: in-progress\nblocked-by: []\n---\n\n" +
				"# T-05 — X\n\nstatus: decoy stays untouched\n",
		},
		{
			name: "inline comment after the value survives byte for byte",
			content: "---\nid: T-01\ntitle: Y\nstatus: in-progress   # todo | done\n" +
				"blocked-by: []\n---\n",
			status: "done",
			want: "---\nid: T-01\ntitle: Y\nstatus: done   # todo | done\n" +
				"blocked-by: []\n---\n",
		},
		{
			name:    "blocked",
			content: "---\nid: T-03\ntitle: Z\nstatus: done\nblocked-by: []\n---\n",
			status:  "blocked",
			want:    "---\nid: T-03\ntitle: Z\nstatus: blocked\nblocked-by: []\n---\n",
		},
		{
			name:    "no status line",
			content: "---\nid: T-03\ntitle: Z\nblocked-by: []\n---\n",
			status:  "done",
			wantErr: true,
		},
		{
			name:    "invalid status rejected",
			content: "---\nid: T-03\ntitle: Z\nstatus: todo\nblocked-by: []\n---\n",
			status:  "bogus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := task.SetStatus([]byte(tt.content), tt.status)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetStatus() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SetStatus() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("SetStatus() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestAppendReport(t *testing.T) {
	tests := []struct {
		name    string
		content string
		appends []string
		want    string
	}{
		{
			name:    "existing heading gets the text below it",
			content: "## Gates\n- [x] built\n\n## Report\n",
			appends: []string{"All gates green.\n"},
			want:    "## Gates\n- [x] built\n\n## Report\n\nAll gates green.\n",
		},
		{
			name:    "two appends accumulate in order",
			content: "## Report\n",
			appends: []string{"first block\n", "second block\n"},
			want:    "## Report\n\nfirst block\n\nsecond block\n",
		},
		{
			name:    "missing heading is created once at the end",
			content: "# T-01 — X\n\n## Gates\n- [ ] check\n",
			appends: []string{"blocked: waiting on T-03\n"},
			want:    "# T-01 — X\n\n## Gates\n- [ ] check\n\n## Report\n\nblocked: waiting on T-03\n",
		},
		{
			name:    "content without trailing newline",
			content: "## Report",
			appends: []string{"note"},
			want:    "## Report\n\nnote\n",
		},
		{
			name:    "empty content grows a report section",
			content: "",
			appends: []string{"orphan report\n"},
			want:    "## Report\n\norphan report\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := []byte(tt.content)
			for _, text := range tt.appends {
				got = task.AppendReport(got, []byte(text))
			}
			if string(got) != tt.want {
				t.Errorf("AppendReport() =\n%q\nwant:\n%q", got, tt.want)
			}
			if n := strings.Count(string(got), "## Report"); n != 1 {
				t.Errorf("heading appears %d times, want 1", n)
			}
		})
	}
}

func TestAppendReportAccumulatesInRenderedFile(t *testing.T) {
	content := task.Render("T-02", "Task file domain package", []string{"T-01"})
	content = task.AppendReport(content, []byte("### first\n\ndetails one\n"))
	content = task.AppendReport(content, []byte("### second\n\ndetails two\n"))

	got := string(content)
	first := strings.Index(got, "### first")
	second := strings.Index(got, "### second")
	heading := strings.Index(got, "## Report")
	if strings.Count(got, "## Report") != 1 {
		t.Errorf("heading count = %d, want 1", strings.Count(got, "## Report"))
	}
	if first == -1 || second == -1 || !(heading < first && first < second) {
		t.Errorf("reports out of order: heading=%d first=%d second=%d\n%s",
			heading, first, second, got)
	}
}

func TestCountGates(t *testing.T) {
	tests := []struct {
		name    string
		content string
		done    int
		total   int
	}{
		{
			name:    "no gates section",
			content: "# T-01 — X\n\n## Goal\nShip it.\n",
		},
		{
			name:    "empty gates section",
			content: "## Gates\n\n## Report\n",
		},
		{
			name:    "all ticked",
			content: "## Gates\n- [x] build\n- [x] test\n\n## Report\n",
			done:    2,
			total:   2,
		},
		{
			name:    "mixed",
			content: "## Gates\n- [x] build\n- [ ] test\n- [ ] vet\n\n## Report\n",
			done:    1,
			total:   3,
		},
		{
			name:    "stops at the next section",
			content: "## Gates\n- [ ] real gate\n\n## Report\n- [x] not a gate\n- [ ] neither\n",
			done:    0,
			total:   1,
		},
		{
			name:    "checkboxes before the section do not count",
			content: "## Scope\n- [x] scope bullet\n\n## Gates\n- [x] gate\n",
			done:    1,
			total:   1,
		},
		{
			name:    "indented checkboxes do not count",
			content: "## Gates\n- [x] gate\n  - [ ] sub-item\n",
			done:    1,
			total:   1,
		},
		{
			name:    "gates section at end of file without trailing section",
			content: "## Gates\n- [ ] one\n- [x] two",
			done:    1,
			total:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, total := task.CountGates([]byte(tt.content))
			if done != tt.done || total != tt.total {
				t.Errorf("CountGates() = %d/%d, want %d/%d", done, total, tt.done, tt.total)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple lowercase",
			title: "short title",
			want:  "short-title",
		},
		{
			name:  "uppercase folded",
			title: "CLI Dispatch",
			want:  "cli-dispatch",
		},
		{
			name:  "punctuation runs collapse to one hyphen",
			title: "fix -- the login!! now",
			want:  "fix-the-login-now",
		},
		{
			name:  "leading and trailing junk trimmed",
			title: "  ...ship it???  ",
			want:  "ship-it",
		},
		{
			name:  "digits survive",
			title: "port to go 1.26",
			want:  "port-to-go-1-26",
		},
		{
			name:  "non-ascii becomes hyphen",
			title: "tâche déjà vue",
			want:  "t-che-d-j-vue",
		},
		{
			name:  "truncated to 40 without trailing hyphen",
			title: "the quick brown fox jumps over the lazy dog",
			want:  "the-quick-brown-fox-jumps-over-the-lazy",
		},
		{
			name:  "exactly 40 kept",
			title: strings.Repeat("a", 40),
			want:  strings.Repeat("a", 40),
		},
		{
			name:  "over 40 same-run truncated",
			title: strings.Repeat("a", 45),
			want:  strings.Repeat("a", 40),
		},
		{
			name:  "all junk yields empty",
			title: "!!! ??? ...",
			want:  "",
		},
		{
			name:  "empty title",
			title: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := task.Slugify(tt.title); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestValidStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{task.StatusTodo, true},
		{task.StatusInProgress, true},
		{task.StatusDone, true},
		{task.StatusBlocked, true},
		{"", false},
		{"Done", false},
		{"todo ", false},
		{"in progress", false},
		{"bogus", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := task.ValidStatus(tt.status); got != tt.want {
				t.Errorf("ValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestBody(t *testing.T) {
	tests := []struct {
		name, content, want string
	}{
		{
			name:    "frontmatter stripped",
			content: "---\nid: T-01\nstatus: todo\n---\n\n# T-01 — Title\n\n## Goal\n",
			want:    "\n# T-01 — Title\n\n## Goal\n",
		},
		{
			name:    "no frontmatter returned whole",
			content: "# Just markdown\n",
			want:    "# Just markdown\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(task.Body([]byte(tt.content))); got != tt.want {
				t.Errorf("Body() = %q, want %q", got, tt.want)
			}
		})
	}
}

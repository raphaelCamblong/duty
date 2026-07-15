package tests

import (
	"strings"
	"testing"

	"github.com/raphaelCamblong/duty/internal/task"
)

// sectionDoc is a task body salted with prose in every section, so a writer
// that touches more than its target section fails a full-string comparison.
const sectionDoc = "---\nid: T-05\ntitle: X\nstatus: todo\nblocked-by: []\n---\n\n" +
	"# T-05 — X\n\n" +
	"## Goal\nThe original goal.\n\n" +
	"## Read first\nRead the spec.\n\n" +
	"## Scope\nDo the thing.\n\n" +
	"## Gates\n- [ ] one\n- [x] two\n\n" +
	"## Report\n\nEarlier report.\n"

func TestSection(t *testing.T) {
	tests := []struct {
		name    string
		heading string
		want    string
		wantOK  bool
	}{
		{name: "goal body", heading: "Goal", want: "The original goal.\n\n", wantOK: true},
		{name: "case-insensitive heading", heading: "gOaL", want: "The original goal.\n\n", wantOK: true},
		{name: "gates body stops at next heading", heading: "Gates", want: "- [ ] one\n- [x] two\n\n", wantOK: true},
		{name: "report body runs to end of file", heading: "Report", want: "\nEarlier report.\n", wantOK: true},
		{name: "missing section", heading: "Design", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, ok := task.Section([]byte(sectionDoc), tt.heading)
			if ok != tt.wantOK {
				t.Fatalf("Section ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && string(body) != tt.want {
				t.Errorf("Section body = %q, want %q", body, tt.want)
			}
		})
	}
}

func TestReplaceSection(t *testing.T) {
	tests := []struct {
		name    string
		content string
		heading string
		body    string
		want    string
		wantErr bool
	}{
		{
			name:    "replaces one section body, every other byte survives",
			content: sectionDoc,
			heading: "Goal",
			body:    "A brand new goal.\n",
			want:    replaceOnce(t, sectionDoc, "## Goal\nThe original goal.\n", "## Goal\nA brand new goal.\n"),
		},
		{
			name:    "case-insensitive match keeps the heading casing",
			content: sectionDoc,
			heading: "scope",
			body:    "Rescoped.",
			want:    replaceOnce(t, sectionDoc, "## Scope\nDo the thing.\n", "## Scope\nRescoped.\n"),
		},
		{
			name:    "multi-line body trimmed and re-emitted",
			content: sectionDoc,
			heading: "Goal",
			body:    "\n\nLine one.\nLine two.\n\n\n",
			want:    replaceOnce(t, sectionDoc, "## Goal\nThe original goal.\n", "## Goal\nLine one.\nLine two.\n"),
		},
		{
			name:    "missing section is created before Report",
			content: sectionDoc,
			heading: "Design",
			body:    "Ports and adapters.",
			want:    replaceOnce(t, sectionDoc, "## Report\n", "## Design\nPorts and adapters.\n\n## Report\n"),
		},
		{
			name:    "missing section with no Report is appended at end of file",
			content: "# T-01 — X\n\n## Goal\nG.\n",
			heading: "Notes",
			body:    "A note.",
			want:    "# T-01 — X\n\n## Goal\nG.\n\n## Notes\nA note.\n",
		},
		{
			name:    "empty heading is rejected",
			content: sectionDoc,
			heading: "   ",
			body:    "x",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := task.ReplaceSection([]byte(tt.content), tt.heading, []byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReplaceSection = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReplaceSection error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("ReplaceSection =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestReplaceSections(t *testing.T) {
	t.Run("replaces three blocks in one call, bytes outside them survive", func(t *testing.T) {
		payload := "## Goal\nNew goal.\n\n## Scope\nNew scope.\n\n## Design\nPorts.\n"
		got, err := task.ReplaceSections([]byte(sectionDoc), []byte(payload))
		if err != nil {
			t.Fatalf("ReplaceSections error = %v", err)
		}
		want := replaceOnce(t, sectionDoc, "## Goal\nThe original goal.\n", "## Goal\nNew goal.\n")
		want = replaceOnce(t, want, "## Scope\nDo the thing.\n", "## Scope\nNew scope.\n")
		want = replaceOnce(t, want, "## Report\n", "## Design\nPorts.\n\n## Report\n")
		if string(got) != want {
			t.Errorf("ReplaceSections =\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("applies blocks in payload order", func(t *testing.T) {
		got, err := task.ReplaceSections([]byte(sectionDoc), []byte("## Alpha\na\n\n## Beta\nb\n"))
		if err != nil {
			t.Fatalf("ReplaceSections error = %v", err)
		}
		if alpha, beta := strings.Index(string(got), "## Alpha"), strings.Index(string(got), "## Beta"); alpha < 0 || beta < alpha {
			t.Errorf("want ## Alpha before ## Beta, got %q", got)
		}
	})

	t.Run("rejects a payload that does not open at a heading", func(t *testing.T) {
		for _, payload := range []string{"", "prose\n## Goal\nx\n", "no heading here\n"} {
			if _, err := task.ReplaceSections([]byte(sectionDoc), []byte(payload)); err == nil {
				t.Errorf("ReplaceSections(%q) = nil error, want refusal", payload)
			}
		}
	})
}

func TestGates(t *testing.T) {
	gates := task.Gates([]byte(sectionDoc))
	want := []task.Gate{{Text: "one", Done: false}, {Text: "two", Done: true}}
	if len(gates) != len(want) {
		t.Fatalf("Gates() = %v, want %v", gates, want)
	}
	for i, g := range gates {
		if g != want[i] {
			t.Errorf("Gates()[%d] = %+v, want %+v", i, g, want[i])
		}
	}
	if got := task.Gates([]byte("# no gates here\n")); got != nil {
		t.Errorf("Gates(no section) = %v, want nil", got)
	}
}

func TestAddGate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		text    string
		want    string
	}{
		{
			name:    "appends after the last gate, existing lines untouched",
			content: sectionDoc,
			text:    "three",
			want:    replaceOnce(t, sectionDoc, "- [x] two\n", "- [x] two\n- [ ] three\n"),
		},
		{
			name:    "trims the gate text",
			content: sectionDoc,
			text:    "  spaced  ",
			want:    replaceOnce(t, sectionDoc, "- [x] two\n", "- [x] two\n- [ ] spaced\n"),
		},
		{
			name:    "creates the Gates section before Report when absent",
			content: "# T-01 — X\n\n## Goal\nG.\n\n## Report\n",
			text:    "first gate",
			want:    "# T-01 — X\n\n## Goal\nG.\n\n## Gates\n- [ ] first gate\n\n## Report\n",
		},
		{
			name:    "fills an existing but empty Gates section",
			content: "## Gates\n\n## Report\n",
			text:    "only",
			want:    "## Gates\n- [ ] only\n\n## Report\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(task.AddGate([]byte(tt.content), tt.text)); got != tt.want {
				t.Errorf("AddGate =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestSetGate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		n       int
		done    bool
		want    string
		wantErr bool
	}{
		{
			name:    "ticks the n-th gate, flipping one byte",
			content: sectionDoc,
			n:       1,
			done:    true,
			want:    replaceOnce(t, sectionDoc, "- [ ] one", "- [x] one"),
		},
		{
			name:    "unticks the n-th gate, flipping one byte",
			content: sectionDoc,
			n:       2,
			done:    false,
			want:    replaceOnce(t, sectionDoc, "- [x] two", "- [ ] two"),
		},
		{
			name:    "index past the last gate errors",
			content: sectionDoc,
			n:       3,
			done:    true,
			wantErr: true,
		},
		{
			name:    "no Gates section errors",
			content: "# T-01 — X\n\n## Goal\nG.\n",
			n:       1,
			done:    true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := task.SetGate([]byte(tt.content), tt.n, tt.done)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetGate = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SetGate error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("SetGate =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

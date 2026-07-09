// Package task models the duty task file format: frontmatter parsing,
// new-task template rendering, targeted line edits (status, report), and
// gate counting. The package is pure — bytes in, bytes out — and never
// touches the filesystem.
package task

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Statuses a task can hold. Any transition between them is legal (a flat
// setter); the workflow discipline lives in the lifecycle contract, not in
// a state machine.
const (
	// StatusTodo marks a task not started yet.
	StatusTodo = "todo"
	// StatusInProgress marks a task a worker has picked up.
	StatusInProgress = "in-progress"
	// StatusDone marks a task whose gates all passed.
	StatusDone = "done"
	// StatusBlocked marks a task stopped on a named missing input.
	StatusBlocked = "blocked"
)

// Task is the machine-owned frontmatter of a task file. Everything below
// the frontmatter is freeform markdown that tooling appends to but never
// rewrites.
type Task struct {
	// ID is the tree-wide unique task id, e.g. "T-07".
	ID string `yaml:"id"`
	// Title is the short imperative title.
	Title string `yaml:"title"`
	// Status is one of the Status… constants.
	Status string `yaml:"status"`
	// BlockedBy lists ids of tasks that must be done first.
	BlockedBy []string `yaml:"blocked-by"`
}

var (
	frontmatterRE = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)
	statusLineRE  = regexp.MustCompile(`(?m)^status: \S+`)
	reportHeadRE  = regexp.MustCompile(`(?m)^## Report[ \t]*\r?$`)
)

// Parse extracts the frontmatter of a task file. It returns an error when
// the content does not open with a ----delimited frontmatter block or when
// that block is not valid YAML.
func Parse(content []byte) (Task, error) {
	m := frontmatterRE.FindSubmatch(content)
	if m == nil {
		return Task{}, errors.New("missing frontmatter")
	}
	var t Task
	if err := yaml.Unmarshal(m[1], &t); err != nil {
		return Task{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	return t, nil
}

// Body returns the markdown below the frontmatter block; content without
// frontmatter is returned whole.
func Body(content []byte) []byte {
	if loc := frontmatterRE.FindIndex(content); loc != nil {
		return content[loc[1]:]
	}
	return content
}

// Render produces a brand-new task file: frontmatter (status todo) plus the
// full section skeleton from the spec. Gates starts as an empty checklist,
// so a fresh task counts 0/0 gates.
func Render(id, title string, blockedBy []string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "---\nid: %s\ntitle: %s\nstatus: %s\nblocked-by: [%s]\n---\n\n",
		id, yamlScalar(title), StatusTodo, strings.Join(blockedBy, ", "))
	fmt.Fprintf(&b, "# %s — %s\n", id, title)
	for _, section := range []string{"Goal", "Read first", "Scope", "Out of scope", "Gates", "Report"} {
		fmt.Fprintf(&b, "\n## %s\n", section)
	}
	return b.Bytes()
}

// SetStatus rewrites the first `status:` line to the given status and leaves
// every other byte untouched. It returns an error for a status outside the
// known set or for content without a status line.
func SetStatus(content []byte, status string) ([]byte, error) {
	if !ValidStatus(status) {
		return nil, fmt.Errorf("invalid status %q", status)
	}
	loc := statusLineRE.FindIndex(content)
	if loc == nil {
		return nil, errors.New("no status line")
	}
	out := make([]byte, 0, len(content)+len(status))
	out = append(out, content[:loc[0]]...)
	out = append(out, "status: "...)
	out = append(out, status...)
	out = append(out, content[loc[1]:]...)
	return out, nil
}

// AppendReport appends text under the ## Report heading, creating the heading
// at the end of the file when missing. Reports accumulate: existing content is
// never rewritten, each call appends one blank-line-separated block.
func AppendReport(content, text []byte) []byte {
	var b bytes.Buffer
	b.Write(content)
	if n := len(content); n > 0 && content[n-1] != '\n' {
		b.WriteByte('\n')
	}
	if !reportHeadRE.Match(content) {
		if b.Len() > 0 && !endsBlank(b.Bytes()) {
			b.WriteByte('\n')
		}
		b.WriteString("## Report\n")
	}
	if b.Len() > 0 && !endsBlank(b.Bytes()) {
		b.WriteByte('\n')
	}
	b.Write(text)
	if n := len(text); n > 0 && text[n-1] != '\n' {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// endsBlank reports whether b ends with a blank line.
func endsBlank(b []byte) bool {
	return len(b) >= 2 && b[len(b)-1] == '\n' && b[len(b)-2] == '\n'
}

// CountGates counts gate checkboxes under the first ## Gates heading,
// stopping at the next ## heading: done is the number of ticked `- [x]`
// lines, total additionally includes the unticked `- [ ]` lines.
func CountGates(content []byte) (done, total int) {
	inGates := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if inGates && strings.HasPrefix(line, "## ") {
			break
		}
		if !inGates {
			inGates = line == "## Gates"
			continue
		}
		switch {
		case strings.HasPrefix(line, "- [x]"):
			done++
			total++
		case strings.HasPrefix(line, "- [ ]"):
			total++
		}
	}
	return done, total
}

// Section returns the trimmed body of the first "## <heading>" section,
// stopping at the next "## " heading; "" when the section is absent or empty.
func Section(content []byte, heading string) string {
	want := "## " + heading
	inSection := false
	var b strings.Builder
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if inSection {
			if strings.HasPrefix(line, "## ") {
				break
			}
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		if line == want {
			inSection = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Slugify derives a filename slug from a title: lowercased, every run of
// non-alphanumeric characters collapsed to one hyphen, no leading or trailing
// hyphen, at most 40 characters. Only ASCII letters and digits survive.
func Slugify(title string) string {
	var b strings.Builder
	pending := false
	for _, r := range strings.ToLower(title) {
		alnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !alnum {
			pending = b.Len() > 0
			continue
		}
		if pending {
			b.WriteByte('-')
			pending = false
		}
		b.WriteRune(r)
	}
	s := b.String()
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	return s
}

// ValidStatus reports whether s is one of the four task statuses.
func ValidStatus(s string) bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone, StatusBlocked:
		return true
	}
	return false
}

// yamlScalar renders s as a one-line YAML scalar: plain when unambiguous,
// double-quoted otherwise. It only ever generates fresh frontmatter —
// existing frontmatter is never re-serialized.
func yamlScalar(s string) string {
	if plainSafe(s) {
		return s
	}
	return strconv.Quote(s)
}

// plainSafe reports whether s survives verbatim as a YAML plain scalar in a
// block-mapping value position. The check is conservative: quoting a plain
// string is always safe, the reverse is not.
func plainSafe(s string) bool {
	if s == "" || s != strings.TrimSpace(s) {
		return false
	}
	if strings.ContainsAny(s, "\n\r\t") {
		return false
	}
	if strings.ContainsAny(s[:1], "-?:,[]{}#&*!|>'\"%@`") {
		return false
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") || strings.Contains(s, " #") {
		return false
	}
	return true
}

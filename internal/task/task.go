// Package task models the duty task file format: frontmatter parsing,
// new-task template rendering, targeted line edits (status, report), and
// gate counting. The package is pure — bytes in, bytes out — and never
// touches the filesystem.
package task

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

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

// IDPrefix is the fixed prefix of every task id and task filename, before the
// zero-padded number.
const IDPrefix = "T-"

// FormatID returns the task id for the zero-padded number nn, e.g. "07" yields
// "T-07".
func FormatID(nn string) string {
	return IDPrefix + nn
}

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

//go:embed task.md.tmpl
var skeletonTmplText string

// skeletonTmpl renders a fresh task file from its embedded template.
var skeletonTmpl = template.Must(template.New("task").Funcs(template.FuncMap{
	"yamlTitle": yamlScalar,
	"blockedBy": func(ids []string) string { return strings.Join(ids, ", ") },
}).Parse(skeletonTmplText))

// Render produces a brand-new task file: frontmatter (status todo) plus the
// full section skeleton from the spec. Gates starts as an empty checklist,
// so a fresh task counts 0/0 gates.
func Render(id, title string, blockedBy []string) []byte {
	var b bytes.Buffer
	_ = skeletonTmpl.Execute(&b, Task{ID: id, Title: title, Status: StatusTodo, BlockedBy: blockedBy})
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
	writeEndingNL(&b, content)
	if !reportHeadRE.Match(content) {
		ensureBlankLine(&b)
		b.WriteString("## Report\n")
	}
	ensureBlankLine(&b)
	writeEndingNL(&b, text)
	return b.Bytes()
}

// writeEndingNL writes p to b, adding a trailing newline when p is non-empty
// and not already newline-terminated.
func writeEndingNL(b *bytes.Buffer, p []byte) {
	b.Write(p)
	if n := len(p); n > 0 && p[n-1] != '\n' {
		b.WriteByte('\n')
	}
}

// ensureBlankLine writes a newline to b when its content is non-empty and does
// not already end on a blank line, so the next block reads blank-line separated.
func ensureBlankLine(b *bytes.Buffer) {
	if b.Len() > 0 && !endsBlank(b.Bytes()) {
		b.WriteByte('\n')
	}
}

// endsBlank reports whether b ends with a blank line.
func endsBlank(b []byte) bool {
	return len(b) >= 2 && b[len(b)-1] == '\n' && b[len(b)-2] == '\n'
}

// CountGates counts gate checkboxes under the "## Gates" section: done is the
// number of ticked gates, total every gate.
func CountGates(content []byte) (done, total int) {
	for _, g := range Gates(content) {
		total++
		if g.Done {
			done++
		}
	}
	return done, total
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

// ValidSlug reports whether s is a slug of the shape Slugify produces: a
// non-empty run of at most 40 lowercase letters, digits, and hyphens, with no
// leading or trailing hyphen.
func ValidSlug(s string) bool {
	if s == "" || len(s) > 40 || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
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

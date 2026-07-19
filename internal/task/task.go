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
	"time"

	"gopkg.in/yaml.v3"
)

// Statuses a task can hold; any transition is legal (a flat setter, no state machine).
const (
	StatusTodo       = "todo"
	StatusInProgress = "in-progress"
	// StatusDone marks a task whose gates all passed.
	StatusDone = "done"
	// StatusBlocked marks a task stopped on a named missing input.
	StatusBlocked = "blocked"
	// StatusBacklog marks a task parked out of the actionable queue until
	// groomed; get next never offers it.
	StatusBacklog = "backlog"
)

// IDPrefix is the fixed prefix of every task id and task filename, before the
// zero-padded number.
const IDPrefix = "T-"

// FormatID returns the task id for the zero-padded number nn, e.g. "07" yields
// "T-07".
func FormatID(nn string) string {
	return IDPrefix + nn
}

// Task is the machine-owned frontmatter of a task file; the markdown below is
// freeform — appended to, never rewritten.
type Task struct {
	// ID is the tree-wide unique task id, e.g. "T-07".
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
	// BlockedBy lists ids of tasks that must be done first.
	BlockedBy []string `yaml:"blocked-by"`
	// ClaimedBy names the agent holding an in-progress task, empty when unclaimed;
	// machine-owned and written line-surgically, so it is absent from fresh templates.
	ClaimedBy string `yaml:"claimed-by,omitempty"`
}

var (
	frontmatterRE = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)
	statusLineRE  = regexp.MustCompile(`(?m)^status: \S+`)
	claimedLineRE = regexp.MustCompile(`(?m)^claimed-by: .*\r?\n`)
)

func Parse(content []byte) (Task, error) {
	match := frontmatterRE.FindSubmatch(content)
	if match == nil {
		return Task{}, errors.New("missing frontmatter")
	}
	var task Task
	if err := yaml.Unmarshal(match[1], &task); err != nil {
		return Task{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	return task, nil
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

var skeletonTmpl = template.Must(template.New("task").Funcs(template.FuncMap{
	"yamlTitle": yamlScalar,
	"blockedBy": func(ids []string) string { return strings.Join(ids, ", ") },
}).Parse(skeletonTmplText))

// Render produces a brand-new task file: frontmatter (status todo) plus the
// empty section skeleton — a fresh task counts 0/0 gates.
func Render(id, title string, blockedBy []string) []byte {
	var buf bytes.Buffer
	_ = skeletonTmpl.Execute(&buf, Task{ID: id, Title: title, Status: StatusTodo, BlockedBy: blockedBy})
	return buf.Bytes()
}

// RenderWithBody produces a brand-new task file: frontmatter (status todo) and H1,
// then body spliced verbatim — byte-identical below the H1 to a section-by-section fill.
func RenderWithBody(id, title string, blockedBy []string, body []byte) []byte {
	head := Render(id, title, blockedBy)
	head = head[:nextHeadingFrom(head, 0)]
	var buf bytes.Buffer
	buf.Write(head)
	writeEndingNL(&buf, bytes.TrimLeft(body, " \t\r\n"))
	return buf.Bytes()
}

// SetStatus rewrites the first status: line and leaves every other byte
// untouched; errors on an unknown status or content with no status line.
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

// SetClaimedBy sets the claimed-by line to name line-surgically (empty name
// removes it), so a claim then clear restores the file byte-for-byte.
func SetClaimedBy(content []byte, name string) ([]byte, error) {
	if loc := claimedLineRE.FindIndex(content); loc != nil {
		return spliceClaim(content, loc[0], loc[1], name), nil
	}
	if name == "" {
		return content, nil
	}
	at, err := afterStatusLine(content)
	if err != nil {
		return nil, err
	}
	return spliceClaim(content, at, at, name), nil
}

func spliceClaim(content []byte, start, end int, name string) []byte {
	var line string
	if name != "" {
		line = "claimed-by: " + yamlScalar(name) + "\n"
	}
	out := make([]byte, 0, len(content)-(end-start)+len(line))
	out = append(out, content[:start]...)
	out = append(out, line...)
	out = append(out, content[end:]...)
	return out
}

func afterStatusLine(content []byte) (int, error) {
	loc := statusLineRE.FindIndex(content)
	if loc == nil {
		return 0, errors.New("no status line")
	}
	nl := bytes.IndexByte(content[loc[1]:], '\n')
	if nl < 0 {
		return 0, errors.New("no status line")
	}
	return loc[1] + nl + 1, nil
}

func ReportHeading(at time.Time, status string) string {
	heading := "### " + at.Format("2006-01-02 15:04")
	if status != "" {
		heading += " — " + status
	}
	return heading
}

func ReportBlock(at time.Time, status string, text []byte) []byte {
	return append([]byte(ReportHeading(at, status)+"\n\n"), text...)
}

// AppendReport appends text as a blank-line-separated block under ## Report
// (created at end of file when missing); existing content is never rewritten.
func AppendReport(content, text []byte) []byte {
	var buf bytes.Buffer
	writeEndingNL(&buf, content)
	if _, ok := headingIndex(content, reportHeading); !ok {
		ensureBlankLine(&buf)
		buf.WriteString("## Report\n")
	}
	ensureBlankLine(&buf)
	writeEndingNL(&buf, text)
	return buf.Bytes()
}

func writeEndingNL(buf *bytes.Buffer, data []byte) {
	buf.Write(data)
	if length := len(data); length > 0 && data[length-1] != '\n' {
		buf.WriteByte('\n')
	}
}

func ensureBlankLine(buf *bytes.Buffer) {
	if buf.Len() > 0 && !endsBlank(buf.Bytes()) {
		buf.WriteByte('\n')
	}
}

func endsBlank(data []byte) bool {
	return len(data) >= 2 && data[len(data)-1] == '\n' && data[len(data)-2] == '\n'
}

func CountGates(content []byte) (done, total int) {
	start, end, found := sectionBounds(content, gatesHeading)
	if !found {
		return 0, 0
	}
	for pos := start; pos < end; {
		line, next := lineAt(content, pos)
		switch {
		case bytes.HasPrefix(line, []byte("- [x]")):
			done++
			total++
		case bytes.HasPrefix(line, []byte("- [ ]")):
			total++
		}
		pos = next
	}
	return done, total
}

const maxSlugLen = 40

// Slugify derives a filename slug from title: lowercased, non-alphanumeric runs
// collapsed to single hyphens, no leading or trailing hyphen, capped at
// maxSlugLen on a word boundary.
func Slugify(title string) string {
	var builder strings.Builder
	pending := false
	for _, char := range strings.ToLower(title) {
		alnum := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if !alnum {
			pending = builder.Len() > 0
			continue
		}
		if pending {
			builder.WriteByte('-')
			pending = false
		}
		builder.WriteRune(char)
	}
	slug := builder.String()
	if len(slug) > maxSlugLen {
		slug = truncateSlug(slug)
	}
	return slug
}

func truncateSlug(slug string) string {
	cut := slug[:maxSlugLen]
	if dashIdx := strings.LastIndexByte(cut, '-'); dashIdx > 0 {
		return cut[:dashIdx]
	}
	return strings.TrimRight(cut, "-")
}

// ValidSlug reports whether slug has the shape Slugify produces: 1..maxSlugLen
// chars of [a-z0-9-], no leading or trailing hyphen.
func ValidSlug(slug string) bool {
	if slug == "" || len(slug) > maxSlugLen || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for _, char := range slug {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return true
}

func ValidStatus(status string) bool {
	switch status {
	case StatusTodo, StatusInProgress, StatusDone, StatusBlocked, StatusBacklog:
		return true
	}
	return false
}

// yamlScalar renders value as a one-line YAML scalar (plain when unambiguous,
// double-quoted otherwise); it only generates fresh frontmatter, never re-serializes.
func yamlScalar(value string) string {
	if plainSafe(value) {
		return value
	}
	return strconv.Quote(value)
}

// plainSafe reports whether value survives verbatim as a YAML plain scalar in a
// block-mapping value; the check is conservative (a false negative only over-quotes).
func plainSafe(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	if strings.ContainsAny(value, "\n\r\t") {
		return false
	}
	if strings.ContainsAny(value[:1], "-?:,[]{}#&*!|>'\"%@`") {
		return false
	}
	if strings.Contains(value, ": ") || strings.HasSuffix(value, ":") || strings.Contains(value, " #") {
		return false
	}
	return true
}

package app

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/raphaelCamblong/duty/internal/fetch"
	"github.com/raphaelCamblong/duty/internal/names"
)

//go:embed skill.md
var skillText []byte

// Target names an agent harness the skill installs into.
type Target string

const (
	Claude Target = "claude" // .claude/skills/duty/SKILL.md
	Codex  Target = "codex"  // a delimited block in AGENTS.md
	Gemini Target = "gemini" // a delimited block in GEMINI.md
)

// The HTML markers that delimit the duty block in AGENTS.md / GEMINI.md, so a
// later install finds and replaces exactly that block.
const (
	skillBlockStart = "<!-- duty:skill start -->"
	skillBlockEnd   = "<!-- duty:skill end -->"
)

func unknownTargetErr(s string) error {
	return fmt.Errorf("unknown target %q: want claude, codex or gemini", s)
}

func ParseTarget(s string) (Target, error) {
	switch Target(s) {
	case Claude, Codex, Gemini:
		return Target(s), nil
	}
	return "", unknownTargetErr(s)
}

// Skill returns the agent skill text: the copy fetched from url when reachable,
// otherwise the embedded fallback. offline (or a nil fetcher) skips the network
// entirely; any fetch failure or empty body falls back silently.
func (a App) Skill(f fetch.Fetcher, url string, offline bool) []byte {
	if offline || f == nil {
		return skillText
	}
	body, err := f.Fetch(url)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		return skillText
	}
	return body
}

// InstallSkill writes skill for target: Claude as a standalone SKILL.md (project
// scope under cwd, or home scope with user), Codex and Gemini as a marker-
// delimited block in AGENTS.md / GEMINI.md at cwd. Without force it refuses to
// overwrite an existing skill; with force it replaces it cleanly. It returns the
// path written.
func (a App) InstallSkill(cwd, home string, target Target, skill []byte, user, force bool) (string, error) {
	if user && target != Claude {
		return "", errors.New("--user applies only to the claude target")
	}
	switch target {
	case Claude:
		return a.installClaude(cwd, home, skill, user, force)
	case Codex:
		return a.installBlock(filepath.Join(cwd, names.AgentsFile), skill, force)
	case Gemini:
		return a.installBlock(filepath.Join(cwd, names.GeminiFile), skill, force)
	}
	return "", unknownTargetErr(string(target))
}

func (a App) installClaude(cwd, home string, skill []byte, user, force bool) (string, error) {
	base := cwd
	if user {
		if home == "" {
			return "", errors.New("cannot resolve the home directory for --user")
		}
		base = home
	}
	dir := filepath.Join(base, names.ClaudeDir, names.SkillsDir, names.SkillName)
	path := filepath.Join(dir, names.SkillFile)
	if !force {
		if _, err := a.fs.Stat(path); err == nil {
			return "", fmt.Errorf("%s already exists (use --force to replace)", path)
		}
	}
	if err := a.fs.MkdirAll(dir); err != nil {
		return "", err
	}
	if err := a.fs.WriteFile(path, skill); err != nil {
		return "", err
	}
	return path, nil
}

func (a App) installBlock(path string, skill []byte, force bool) (string, error) {
	existing, err := a.fs.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if hasSkillBlock(existing) && !force {
		return "", fmt.Errorf("%s already has a duty skill block (use --force to replace)", path)
	}
	if err := a.fs.WriteFile(path, mergeSkillBlock(existing, skill)); err != nil {
		return "", err
	}
	return path, nil
}

func hasSkillBlock(content []byte) bool {
	return bytes.Contains(content, []byte(skillBlockStart))
}

// skillBlock wraps skill's body (frontmatter stripped) in the duty markers,
// ending with a single newline.
func skillBlock(skill []byte) string {
	body := strings.TrimRight(string(skillBody(skill)), "\n")
	return skillBlockStart + "\n" + body + "\n" + skillBlockEnd + "\n"
}

// mergeSkillBlock returns existing with the duty block replaced in place, or
// appended below a blank line when existing carries no block.
func mergeSkillBlock(existing, skill []byte) []byte {
	block := skillBlock(skill)
	s := string(existing)
	start := strings.Index(s, skillBlockStart)
	if start < 0 {
		return []byte(separate(s) + block)
	}
	after := ""
	if e := strings.Index(s[start:], skillBlockEnd); e >= 0 {
		after = strings.TrimPrefix(s[start+e+len(skillBlockEnd):], "\n")
	}
	return []byte(s[:start] + block + after)
}

// separate ends s with a blank-line separator so an appended block is set off,
// leaving an empty file empty.
func separate(s string) string {
	switch {
	case s == "":
		return ""
	case strings.HasSuffix(s, "\n\n"):
		return s
	case strings.HasSuffix(s, "\n"):
		return s + "\n"
	default:
		return s + "\n\n"
	}
}

// skillBody returns skill with any leading YAML frontmatter removed, so the
// marker-delimited harnesses embed the prose without the Claude header.
func skillBody(skill []byte) []byte {
	const fence = "---\n"
	s := string(skill)
	if !strings.HasPrefix(s, fence) {
		return skill
	}
	i := strings.Index(s[len(fence):], "\n---")
	if i < 0 {
		return skill
	}
	rest := s[len(fence)+i+len("\n---"):]
	return []byte(strings.TrimLeft(rest, "\n"))
}

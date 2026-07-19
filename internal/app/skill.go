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

func unknownTargetErr(raw string) error {
	return fmt.Errorf("unknown target %q: want claude, codex or gemini", raw)
}

func ParseTarget(raw string) (Target, error) {
	switch Target(raw) {
	case Claude, Codex, Gemini:
		return Target(raw), nil
	}
	return "", unknownTargetErr(raw)
}

// Skill returns the agent skill text fetched from url, or the embedded fallback
// when offline, the fetcher is nil, or the fetch fails or is empty.
func (a App) Skill(fetcher fetch.Fetcher, url string, offline bool) []byte {
	if offline || fetcher == nil {
		return skillText
	}
	body, err := fetcher.Fetch(url)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		return skillText
	}
	return body
}

// Install is where and how InstallSkill writes a skill: the target harness, the
// Cwd and Home base dirs it may write under, and the User (home-scope) and Force
// (overwrite) flags.
type Install struct {
	Target Target
	Cwd    string
	Home   string
	User   bool
	Force  bool
}

// InstallSkill writes skill per spec and returns the path: Claude as a standalone
// SKILL.md, Codex/Gemini as a marker block; Force replaces an existing skill.
func (a App) InstallSkill(spec Install, skill []byte) (string, error) {
	if spec.User && spec.Target != Claude {
		return "", errors.New("--user applies only to the claude target")
	}
	switch spec.Target {
	case Claude:
		return a.installClaude(spec, skill)
	case Codex:
		return a.installBlock(filepath.Join(spec.Cwd, names.AgentsFile), skill, spec.Force)
	case Gemini:
		return a.installBlock(filepath.Join(spec.Cwd, names.GeminiFile), skill, spec.Force)
	}
	return "", unknownTargetErr(string(spec.Target))
}

func (a App) installClaude(spec Install, skill []byte) (string, error) {
	base := spec.Cwd
	if spec.User {
		if spec.Home == "" {
			return "", errors.New("cannot resolve the home directory for --user")
		}
		base = spec.Home
	}
	dir := filepath.Join(base, names.ClaudeDir, names.SkillsDir, names.SkillName)
	path := filepath.Join(dir, names.SkillFile)
	if !spec.Force {
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
	text := string(existing)
	start := strings.Index(text, skillBlockStart)
	if start < 0 {
		return []byte(separate(text) + block)
	}
	after := ""
	if end := strings.Index(text[start:], skillBlockEnd); end >= 0 {
		after = strings.TrimPrefix(text[start+end+len(skillBlockEnd):], "\n")
	}
	return []byte(text[:start] + block + after)
}

// separate ends text with a blank-line separator so an appended block is set off,
// leaving an empty file empty.
func separate(text string) string {
	switch {
	case text == "":
		return ""
	case strings.HasSuffix(text, "\n\n"):
		return text
	case strings.HasSuffix(text, "\n"):
		return text + "\n"
	default:
		return text + "\n\n"
	}
}

// skillBody returns skill with any leading YAML frontmatter removed, so the
// marker-delimited harnesses embed the prose without the Claude header.
func skillBody(skill []byte) []byte {
	const fence = "---\n"
	text := string(skill)
	if !strings.HasPrefix(text, fence) {
		return skill
	}
	idx := strings.Index(text[len(fence):], "\n---")
	if idx < 0 {
		return skill
	}
	rest := text[len(fence)+idx+len("\n---"):]
	return []byte(strings.TrimLeft(rest, "\n"))
}

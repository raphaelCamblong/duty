package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raphaelCamblong/duty/internal/app"
	"github.com/raphaelCamblong/duty/internal/fetch"
	"github.com/raphaelCamblong/duty/internal/fsys"
)

// appendToFile appends extra to the file at path, failing the test on error.
func appendToFile(t *testing.T, path, extra string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(extra); err != nil {
		t.Fatalf("append %s: %v", path, err)
	}
}

// embeddedSkill returns the skill text baked into the binary, via the offline
// path that never dials.
func embeddedSkill(t *testing.T) []byte {
	t.Helper()
	return app.New(fsys.NewMem()).Skill(nil, "", true)
}

// recordingFetcher is a Fetcher that records whether it was called; the offline
// path must never call it.
type recordingFetcher struct{ called bool }

func (f *recordingFetcher) Fetch(string) ([]byte, error) {
	f.called = true
	return []byte("MUST-NOT-BE-USED"), nil
}

func TestSkillPrint(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runDuty(t, dir, "skill", "--offline")
	if code != 0 || stderr != "" {
		t.Fatalf("skill --offline: code=%d stderr=%q", code, stderr)
	}
	if !strings.HasPrefix(stdout, "---\nname: duty\n") {
		t.Errorf("skill output does not start with the SKILL.md frontmatter:\n%q", stdout[:min(len(stdout), 60)])
	}
	if []byte(stdout)[len(stdout)-1] != '\n' || !bytes.Equal([]byte(stdout), embeddedSkill(t)) {
		t.Errorf("skill --offline output differs from the embedded copy")
	}
}

func TestSkillTextShape(t *testing.T) {
	s := string(embeddedSkill(t))
	if n := strings.Count(s, "\n"); n > 80 {
		t.Errorf("skill is %d lines, want <= 80", n)
	}
	loop := strings.Index(s, "## The loop")
	rules := strings.Index(s, "## Rules")
	if loop < 0 || rules < 0 || loop > rules {
		t.Fatalf("skill must lead with the loop before the rules (loop=%d rules=%d)", loop, rules)
	}
	for _, call := range []string{
		"duty get next --claim",
		"duty gates check <id> --all",
		"duty report <id> --status done",
	} {
		if i := strings.Index(s, call); i < 0 || i > rules {
			t.Errorf("four-call loop must appear (in order, before the rules): missing %q", call)
		}
	}
	for _, want := range []string{"duty --help", "duty <command> --help", "lists no flags"} {
		if !strings.Contains(s, want) {
			t.Errorf("skill must point at --help: missing %q", want)
		}
	}
	if strings.Contains(s, "\nFlags:") {
		t.Errorf("skill must not enumerate flags in a Flags: list")
	}
}

func TestSkillInstallClaude(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "skills", "duty", "SKILL.md")

	code, stdout, stderr := runDuty(t, dir, "skill", "install", "claude", "--offline")
	if code != 0 || stderr != "" {
		t.Fatalf("install claude: code=%d stderr=%q", code, stderr)
	}
	if want := "installed claude skill → " + path + "\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if got := readText(t, path); !strings.HasPrefix(got, "---\nname: duty\n") {
		t.Errorf("claude SKILL.md missing frontmatter:\n%q", got[:min(len(got), 40)])
	}

	code, _, stderr = runDuty(t, dir, "skill", "install", "claude", "--offline")
	if code != 1 {
		t.Fatalf("second install without --force: code=%d, want 1", code)
	}
	oneLine(t, "stderr", stderr)
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("stderr = %q, want it to say already exists", stderr)
	}

	mustRunOut(t, dir, "skill", "install", "claude", "--offline", "--force")
}

func TestSkillInstallMarkers(t *testing.T) {
	for _, tc := range []struct {
		target, file string
	}{
		{"codex", "AGENTS.md"},
		{"gemini", "GEMINI.md"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.file)

			mustRunOut(t, dir, "skill", "install", tc.target, "--offline")
			got := readText(t, path)
			if !strings.Contains(got, "<!-- duty:skill start -->") || !strings.Contains(got, "<!-- duty:skill end -->") {
				t.Fatalf("%s missing skill markers:\n%q", tc.file, got)
			}
			if strings.Contains(got, "name: duty\ndescription:") {
				t.Errorf("%s should carry the body without YAML frontmatter", tc.file)
			}

			appendToFile(t, path, "\n## Hand-written\nkeep me\n")
			code, _, stderr := runDuty(t, dir, "skill", "install", tc.target, "--offline")
			if code != 1 || !strings.Contains(stderr, "already has a duty skill block") {
				t.Fatalf("reinstall without --force: code=%d stderr=%q", code, stderr)
			}

			mustRunOut(t, dir, "skill", "install", tc.target, "--offline", "--force")
			got = readText(t, path)
			if strings.Count(got, "<!-- duty:skill start -->") != 1 || strings.Count(got, "<!-- duty:skill end -->") != 1 {
				t.Errorf("--force must leave exactly one block:\n%q", got)
			}
			if !strings.Contains(got, "keep me") {
				t.Errorf("--force must preserve hand-written content around the block")
			}
		})
	}
}

func TestSkillInstallUserScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()

	mustRunOut(t, dir, "skill", "install", "claude", "--user", "--offline")
	if got := readText(t, filepath.Join(home, ".claude", "skills", "duty", "SKILL.md")); !strings.HasPrefix(got, "---\nname: duty\n") {
		t.Errorf("--user did not write a SKILL.md into HOME")
	}

	code, _, stderr := runDuty(t, dir, "skill", "install", "codex", "--user", "--offline")
	if code != 1 || !strings.Contains(stderr, "--user applies only to the claude target") {
		t.Errorf("codex --user: code=%d stderr=%q", code, stderr)
	}
}

func TestSkillInstallErrors(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runDuty(t, dir, "skill", "install", "bogus", "--offline")
	if code != 1 || !strings.Contains(stderr, `unknown target "bogus"`) {
		t.Errorf("unknown target: code=%d stderr=%q", code, stderr)
	}

	code, _, stderr = runDuty(t, dir, "skill", "install", "--offline")
	if code != 1 || !strings.Contains(stderr, "name a target") {
		t.Errorf("no target, non-interactive: code=%d stderr=%q", code, stderr)
	}
}

func TestSkillFetch(t *testing.T) {
	embedded := embeddedSkill(t)
	a := app.New(fsys.NewMem())

	t.Run("remote wins", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("REMOTE-SKILL-BODY\n"))
		}))
		defer srv.Close()
		if got := a.Skill(fetch.HTTP{}, srv.URL, false); string(got) != "REMOTE-SKILL-BODY\n" {
			t.Errorf("remote fetch = %q, want the served body", got)
		}
	})

	t.Run("server down falls back to embedded, no error", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		if got := a.Skill(fetch.HTTP{Timeout: 200 * time.Millisecond}, url, false); !bytes.Equal(got, embedded) {
			t.Errorf("down server did not fall back to embedded")
		}
	})

	t.Run("non-200 falls back to embedded", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		defer srv.Close()
		if got := a.Skill(fetch.HTTP{}, srv.URL, false); !bytes.Equal(got, embedded) {
			t.Errorf("404 did not fall back to embedded")
		}
	})

	t.Run("offline never dials", func(t *testing.T) {
		f := &recordingFetcher{}
		got := a.Skill(f, "http://example.invalid/skill.md", true)
		if f.called {
			t.Errorf("--offline dialed the network")
		}
		if !bytes.Equal(got, embedded) {
			t.Errorf("offline did not return the embedded copy")
		}
	})
}

func TestSkillRemoteInstalled(t *testing.T) {
	remote := "---\nname: duty\ndescription: remote\n---\n\n# REMOTE-MARKER\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(remote))
	}))
	defer srv.Close()

	mem := fsys.NewMem()
	a := app.New(mem)
	content := a.Skill(fetch.HTTP{}, srv.URL, false)
	path, err := a.InstallSkill(app.Install{Target: app.Claude, Cwd: "/repo"}, content)
	if err != nil {
		t.Fatalf("install remote: %v", err)
	}
	got, err := mem.ReadFile(path)
	if err != nil {
		t.Fatalf("read installed: %v", err)
	}
	if string(got) != remote {
		t.Errorf("installed content = %q, want the remote skill", got)
	}
}

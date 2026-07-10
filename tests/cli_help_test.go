package tests

import (
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	code, stdout, stderr := runDuty(t, t.TempDir(), "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("--help: code=%d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"duty get next", "duty status <id> in-progress", "tick gate checkboxes",
		"duty status <id> done", "duty archive",
		"Author Commands:", "Work Commands:", "Read Commands:", "Interface Commands:",
		"Examples:",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--help missing %q in:\n%s", want, stdout)
		}
	}
}

func TestCommandExamples(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "create task", args: []string{"create", "task", "--help"}, want: `duty create task "Fix login" --blocked-by T-03`},
		{name: "get next", args: []string{"get", "next", "--help"}, want: "duty get next --agent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runDuty(t, t.TempDir(), tt.args...)
			if code != 0 || stderr != "" {
				t.Fatalf("%v: code=%d stderr=%q", tt.args, code, stderr)
			}
			if !strings.Contains(stdout, "Examples:") {
				t.Errorf("%v: missing Examples section in:\n%s", tt.args, stdout)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("%v: missing example %q in:\n%s", tt.args, tt.want, stdout)
			}
		})
	}
}

func TestTypoSuggestion(t *testing.T) {
	code, stdout, stderr := runDuty(t, t.TempDir(), "creat")
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if want := "unknown command \"creat\" — did you mean \"create\"?\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestVersion(t *testing.T) {
	code, stdout, stderr := runDuty(t, t.TempDir(), "--version")
	if code != 0 || stderr != "" {
		t.Fatalf("--version: code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "version") {
		t.Errorf("stdout = %q, want it to name the version", stdout)
	}
}

func TestCompletion(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			code, stdout, stderr := runDuty(t, t.TempDir(), "completion", shell)
			if code != 0 || stderr != "" {
				t.Fatalf("completion %s: code=%d stderr=%q", shell, code, stderr)
			}
			if stdout == "" {
				t.Errorf("completion %s produced no output", shell)
			}
		})
	}
}

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGateColumnNarrowPriority(t *testing.T) {
	root := tuiTree(t)
	path := filepath.Join(root, "T-01-alpha-task.md")
	content := strings.Replace(readText(t, path), "## Gates\n",
		"## Gates\n\n- [x] built\n- [ ] tested\n", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("wide terminal shows both the gate and age columns", func(t *testing.T) {
		frame := newTUIModelSize(t, root, 120, 35).View().Content
		if !strings.Contains(frame, "1/2") {
			t.Errorf("120-col frame missing the gate column:\n%s", frame)
		}
		if !strings.Contains(frame, "just now") {
			t.Errorf("120-col frame missing the age column:\n%s", frame)
		}
	})

	t.Run("narrow terminal keeps age and drops the gate column", func(t *testing.T) {
		frame := newTUIModelSize(t, root, 80, 24).View().Content
		if !strings.Contains(frame, "just now") {
			t.Errorf("80-col frame dropped the always-on age column:\n%s", frame)
		}
		if strings.Contains(frame, "1/2") {
			t.Errorf("80-col frame still shows the gate column, want it hidden:\n%s", frame)
		}
	})
}

func TestScanCarriesMtime(t *testing.T) {
	root := tuiTree(t)
	path := filepath.Join(root, "T-01-alpha-task.md")
	when := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	r, ok := scanRow(mustScan(t, root), ".", "T-01")
	if !ok {
		t.Fatal("T-01 not in snapshot")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !r.UpdatedAt.Equal(info.ModTime()) {
		t.Errorf("row UpdatedAt = %v, want the file mtime %v", r.UpdatedAt, info.ModTime())
	}
}

func TestAgeColumnToggle(t *testing.T) {
	root := tuiTree(t) // fresh files, so every task row reads "just now"

	t.Run("wide terminal shows it, t hides then restores it", func(t *testing.T) {
		m := newTUIModelSize(t, root, 120, 35)
		if !m.ShowAge() {
			t.Fatal("age column hidden by default at 120 cols, want shown")
		}
		if frame := m.View().Content; !strings.Contains(frame, "just now") {
			t.Errorf("120-col frame missing the age column:\n%s", frame)
		}

		m, _ = press(t, m, "t")
		if m.ShowAge() {
			t.Fatal("t did not hide the age column")
		}
		if frame := m.View().Content; strings.Contains(frame, "just now") {
			t.Errorf("age column still rendered after t:\n%s", frame)
		}

		m, _ = press(t, m, "t")
		if !m.ShowAge() {
			t.Fatal("second t did not restore the age column")
		}
		if frame := m.View().Content; !strings.Contains(frame, "just now") {
			t.Errorf("second t did not bring the age column back:\n%s", frame)
		}
	})

	t.Run("narrow terminal still shows it by default, t toggles it off", func(t *testing.T) {
		m := newTUIModelSize(t, root, 70, 20)
		if !m.ShowAge() {
			t.Fatal("age column hidden by default at 70 cols, want shown (always-on)")
		}
		if frame := m.View().Content; !strings.Contains(frame, "just now") {
			t.Errorf("70-col frame missing the always-on age column:\n%s", frame)
		}
		m, _ = press(t, m, "t")
		if m.ShowAge() {
			t.Error("t did not turn the age column off below 100 cols")
		}
	})
}

package tests

import (
	"testing"
	"time"

	"github.com/raphaelCamblong/duty/internal/humanize"
)

func TestRelTime(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	const day = 24 * time.Hour
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"under a minute reads just now", 30 * time.Second, "just now"},
		{"the last second under a minute", 59 * time.Second, "just now"},
		{"the first second past a minute", 61 * time.Second, "1m ago"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"the last minute of the hour", 59 * time.Minute, "59m ago"},
		{"one hour", time.Hour, "1h ago"},
		{"the last hour of the day", 23 * time.Hour, "23h ago"},
		{"just past a day rounds to days", 25 * time.Hour, "1d ago"},
		{"the last hours under a week", 6*day + 23*time.Hour, "6d ago"},
		{"a full week still reads in days", 7 * day, "7d ago"},
		{"past a week switches to an absolute date", 8 * day, now.Add(-8 * day).Format("2006-01-02")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanize.RelTime(now.Add(-tt.ago), now); got != tt.want {
				t.Errorf("RelTime(now-%s, now) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}

	t.Run("a future instant reads just now, never negative", func(t *testing.T) {
		if got := humanize.RelTime(now.Add(time.Hour), now); got != "just now" {
			t.Errorf("RelTime(future) = %q, want \"just now\"", got)
		}
	})
}

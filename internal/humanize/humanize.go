// Package humanize formats timestamps as short, human-friendly ages. It is a
// leaf: standard library only, imported by the presentation layers (cli, tui)
// so the relative-time rule lives in exactly one place.
package humanize

import (
	"fmt"
	"time"
)

const dateFormat = "2006-01-02"

func RelTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 8*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	}
	return t.Format(dateFormat)
}

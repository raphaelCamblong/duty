// Package humanize formats timestamps as short, human-friendly ages. It is a
// leaf: standard library only, imported by the presentation layers (cli, tui)
// so the relative-time rule lives in exactly one place.
package humanize

import (
	"fmt"
	"time"
)

const dateFormat = "2006-01-02"

func RelTime(when, now time.Time) string {
	elapsed := now.Sub(when)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	case elapsed < 8*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(elapsed.Hours())/24)
	}
	return when.Format(dateFormat)
}

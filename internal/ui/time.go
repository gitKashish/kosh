package ui

import (
	"fmt"
	"strings"
	"time"
)

// RelativeTime formats a time.Time into a human-readable relative time string.
// It displays up to two significant units (e.g., "1 day 2 hours ago").
func RelativeTime(t time.Time, now time.Time) string {
	if t.IsZero() {
		return "never"
	}

	isPast := t.Before(now)

	var diff time.Duration
	if isPast {
		diff = now.Sub(t)
	} else {
		diff = t.Sub(now)
	}

	seconds := int(diff.Seconds())

	if seconds < 10 {
		if isPast {
			return "just now"
		}
		return "in a few seconds"
	}

	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	months := days / 30
	years := days / 365

	var parts []string

	// Helper to safely append pluralized units
	addUnit := func(val int, unit string) {
		parts = append(parts, fmt.Sprintf("%02d%s", val, unit))
	}

	// Determine the two most significant units
	switch {
	case years > 0:
		addUnit(years, "y")
		addUnit((days%365)/30, "M")
	case months > 0:
		addUnit(months, "M")
		addUnit(days%30, "d")
	case days > 0:
		addUnit(days, "d")
		addUnit(hours%24, "h")
	case hours > 0:
		addUnit(hours, "h")
		addUnit(minutes%60, "m")
	case minutes > 0:
		addUnit(minutes, "m")
	default:
		addUnit(seconds, "s")
	}

	if len(parts) == 0 {
		return "just now"
	}

	// Join the two largest units
	res := strings.Join(parts, " ")

	if isPast {
		return res + " ago"
	}
	return "in " + res
}

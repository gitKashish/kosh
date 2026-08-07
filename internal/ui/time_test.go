package ui_test

import (
	"testing"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/ui"
)

// Fixed reference point for every case. Calling time.Now() separately for
// val and now leaves a sub-microsecond gap between them, which is enough to
// push a boundary case across a threshold.
var now = time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)

// ago and ahead build a val the given distance either side of now.
func ago(d time.Duration) time.Time   { return now.Add(-d) }
func ahead(d time.Duration) time.Time { return now.Add(d) }

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		val  time.Time
		now  time.Time
		want string
	}{
		{
			name: "nil/zero time returns never",
			val:  time.Time{},
			now:  now,
			want: "never",
		},
		{
			name: "zero time returns never even with a zero now",
			val:  time.Time{},
			now:  time.Time{},
			want: "never",
		},
		{
			name: "identical instants are treated as future",
			val:  now,
			now:  now,
			want: "in a few seconds",
		},

		// Under ten seconds collapses to a fixed phrase
		{
			name: "less than 10 seconds in past gives 'just now'",
			val:  ago(9 * time.Second),
			now:  now,
			want: "just now",
		},
		{
			name: "fractional seconds truncate below the threshold",
			val:  ago(9500 * time.Millisecond),
			now:  now,
			want: "just now",
		},
		{
			name: "less than 10 seconds in future gives 'in a few seconds'",
			val:  ahead(9 * time.Second),
			now:  now,
			want: "in a few seconds",
		},
		{
			name: "a nanosecond ahead is still 'in a few seconds'",
			val:  ahead(time.Nanosecond),
			now:  now,
			want: "in a few seconds",
		},

		// Seconds
		{
			name: "exactly 10 seconds in past gives exact seconds",
			val:  ago(10 * time.Second),
			now:  now,
			want: "10s ago",
		},
		{
			name: "45 seconds in past gives exact seconds",
			val:  ago(45 * time.Second),
			now:  now,
			want: "45s ago",
		},
		{
			name: "45 seconds in future gives exact seconds",
			val:  ahead(45 * time.Second),
			now:  now,
			want: "in 45s",
		},
		{
			name: "last second before a minute",
			val:  ago(59 * time.Second),
			now:  now,
			want: "59s ago",
		},

		// Minutes are reported alone; the trailing seconds are dropped
		{
			name: "exactly one minute",
			val:  ago(time.Minute),
			now:  now,
			want: "01m ago",
		},
		{
			name: "minute and a half drops the seconds",
			val:  ago(90 * time.Second),
			now:  now,
			want: "01m ago",
		},
		{
			name: "last minute before an hour",
			val:  ago(59 * time.Minute),
			now:  now,
			want: "59m ago",
		},
		{
			name: "minutes in future",
			val:  ahead(20 * time.Minute),
			now:  now,
			want: "in 20m",
		},

		// Hours carry the remaining minutes
		{
			name: "exactly one hour keeps the zero minutes",
			val:  ago(time.Hour),
			now:  now,
			want: "01h 00m ago",
		},
		{
			name: "hours and minutes",
			val:  ago(2*time.Hour + 5*time.Minute),
			now:  now,
			want: "02h 05m ago",
		},
		{
			name: "hours in future",
			val:  ahead(90 * time.Minute),
			now:  now,
			want: "in 01h 30m",
		},

		// Days carry the remaining hours
		{
			name: "exactly one day keeps the zero hours",
			val:  ago(24 * time.Hour),
			now:  now,
			want: "01d 00h ago",
		},
		{
			name: "days and hours",
			val:  ago((3*24 + 7) * time.Hour),
			now:  now,
			want: "03d 07h ago",
		},
		{
			name: "days in future",
			val:  ahead(2 * 24 * time.Hour),
			now:  now,
			want: "in 02d 00h",
		},

		// A month is a flat 30 days
		{
			name: "thirty days becomes one month",
			val:  ago(30 * 24 * time.Hour),
			now:  now,
			want: "01M 00d ago",
		},
		{
			name: "months and days",
			val:  ago(45 * 24 * time.Hour),
			now:  now,
			want: "01M 15d ago",
		},
		{
			name: "several months",
			val:  ago(100 * 24 * time.Hour),
			now:  now,
			want: "03M 10d ago",
		},
		{
			name: "months in future",
			val:  ahead(45 * 24 * time.Hour),
			now:  now,
			want: "in 01M 15d",
		},

		// A year is a flat 365 days, and the trailing month count comes from
		// the leftover days rather than from the month total
		{
			name: "exactly one year",
			val:  ago(365 * 24 * time.Hour),
			now:  now,
			want: "01y 00M ago",
		},
		{
			name: "year and a month",
			val:  ago(400 * 24 * time.Hour),
			now:  now,
			want: "01y 01M ago",
		},
		{
			name: "several years",
			val:  ago(1000 * 24 * time.Hour),
			now:  now,
			want: "02y 09M ago",
		},
		{
			name: "years in future",
			val:  ahead(365 * 24 * time.Hour),
			now:  now,
			want: "in 01y 00M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ui.RelativeTime(tt.val, tt.now)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRelativeTimeUnitHandover pins the points where the output switches from
// one unit to the next, so a change to the thresholds can't slip through.
func TestRelativeTimeUnitHandover(t *testing.T) {
	tests := []struct {
		name string
		days int
		want string
	}{
		{name: "last day reported in days", days: 29, want: "29d 00h ago"},
		{name: "first day reported in months", days: 30, want: "01M 00d ago"},
		{name: "last day reported in months", days: 364, want: "12M 04d ago"},
		{name: "first day reported in years", days: 365, want: "01y 00M ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ui.RelativeTime(ago(time.Duration(tt.days)*24*time.Hour), now)
			if got != tt.want {
				t.Errorf("%d days ago: got %q, want %q", tt.days, got, tt.want)
			}
		})
	}
}

// TestRelativeTimeSymmetry checks that an interval renders the same either
// side of now, differing only in tense.
func TestRelativeTimeSymmetry(t *testing.T) {
	tests := []struct {
		name       string
		gap        time.Duration
		wantPast   string
		wantFuture string
	}{
		{
			name:       "seconds",
			gap:        30 * time.Second,
			wantPast:   "30s ago",
			wantFuture: "in 30s",
		},
		{
			name:       "hours",
			gap:        2*time.Hour + 30*time.Minute,
			wantPast:   "02h 30m ago",
			wantFuture: "in 02h 30m",
		},
		{
			name:       "days",
			gap:        5 * 24 * time.Hour,
			wantPast:   "05d 00h ago",
			wantFuture: "in 05d 00h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.RelativeTime(ago(tt.gap), now); got != tt.wantPast {
				t.Errorf("past: got %q, want %q", got, tt.wantPast)
			}
			if got := ui.RelativeTime(ahead(tt.gap), now); got != tt.wantFuture {
				t.Errorf("future: got %q, want %q", got, tt.wantFuture)
			}
		})
	}
}

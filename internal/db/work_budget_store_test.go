package db

import (
	"testing"
	"time"
)

// TestUTCDayWindowsAtMidnightBoundary pins the budget window key: instants a
// millisecond apart across a UTC midnight land in different windows, and a
// non-UTC instant is keyed by its UTC day, not its local day.
func TestUTCDayWindowsAtMidnightBoundary(t *testing.T) {
	lastInstant := time.Date(2026, 8, 1, 23, 59, 59, 999_000_000, time.UTC)
	firstInstant := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	if utcDay(lastInstant) != "2026-08-01" {
		t.Fatalf("utcDay(23:59:59.999) = %q, want 2026-08-01", utcDay(lastInstant))
	}
	if utcDay(firstInstant) != "2026-08-02" {
		t.Fatalf("utcDay(00:00:00) = %q, want 2026-08-02", utcDay(firstInstant))
	}

	// 2026-08-02 01:30 in UTC+3 is still 2026-08-01 in UTC.
	eastOfGreenwich := time.Date(2026, 8, 2, 1, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	if utcDay(eastOfGreenwich) != "2026-08-01" {
		t.Fatalf("utcDay(east of Greenwich) = %q, want the UTC day 2026-08-01", utcDay(eastOfGreenwich))
	}
}

func TestUTCDayStartIsMidnightUTC(t *testing.T) {
	instant := time.Date(2026, 8, 2, 17, 45, 12, 0, time.UTC)
	start := utcDayStart(instant)
	want := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	if !start.Equal(want) {
		t.Fatalf("utcDayStart = %s, want %s", start, want)
	}
	if utcDay(start) != utcDay(instant) {
		t.Fatalf("day start %q and instant %q disagree on the window", utcDay(start), utcDay(instant))
	}
}

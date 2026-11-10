package store

import (
	"testing"
	"time"
)

func TestWallClockKeepsWibFields(t *testing.T) {
	// A moment read in WIB must keep its WIB wall-clock fields regardless of the
	// server's own timezone, which is what the `timestamp` columns store.
	wib := time.FixedZone("WIB", 7*60*60)
	moment := time.Date(2026, 11, 10, 14, 2, 37, 412086000, wib)

	got := WallClock(moment)
	if got.Hour() != 14 || got.Minute() != 2 {
		t.Fatalf("WallClock = %s, want WIB wall-clock 14:02", got.Format("15:04:05"))
	}
	// It is tagged UTC so pgx renders exactly those fields into the column.
	if _, offset := got.Zone(); offset != 0 {
		t.Fatalf("WallClock must be tagged UTC but has offset %d", offset)
	}
}

func TestNormalizeDoesNotShiftStoredTimestamps(t *testing.T) {
	// pgx hands back a `timestamp` value tagged UTC whose fields already are WIB.
	stored := time.Date(2026, 11, 10, 14, 2, 37, 0, time.UTC)
	if got := normalize(stored); got != "2026-11-10T14:02:37" {
		t.Fatalf("normalize = %v, want the stored wall-clock unchanged", got)
	}
}

func TestNormalizeRendersDatesWithoutTime(t *testing.T) {
	// A `date` column arrives as midnight, the way Java serialises LocalDate.
	stored := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	if got := normalize(stored); got != "2026-12-10" {
		t.Fatalf("normalize = %v, want a bare date", got)
	}
}

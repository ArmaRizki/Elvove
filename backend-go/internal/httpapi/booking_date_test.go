package httpapi

import (
	"testing"
	"time"

	"github.com/elvove/photography/backend-go/internal/store"
)

func TestEventDateRejectsYesterdayInWib(t *testing.T) {
	// The boundary must be midnight WIB, not midnight UTC: time.Truncate(24h)
	// snapped to UTC and let yesterday's date through between 00:00 and 07:00 WIB.
	today := store.WallClock(time.Now())
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := today.AddDate(0, 0, 1).Format("2006-01-02")

	if message := eventDateError(yesterday); message != "Event date must be today or later" {
		t.Fatalf("yesterday (%s) gave %q, want it rejected", yesterday, message)
	}
	if message := eventDateError(today.Format("2006-01-02")); message != "" {
		t.Fatalf("today (%s) was rejected with %q", today.Format("2006-01-02"), message)
	}
	if message := eventDateError(tomorrow); message != "" {
		t.Fatalf("tomorrow (%s) was rejected with %q", tomorrow, message)
	}
}

func TestEventDateMessagesMirrorJava(t *testing.T) {
	cases := map[string]string{
		"":           "Event date is required",
		"bukan-date": "Event date must be a valid date",
		"2026-13-40": "Event date must be a valid date",
	}
	for value, want := range cases {
		if got := eventDateError(value); got != want {
			t.Errorf("eventDateError(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestTodayWibIsMidnightWib(t *testing.T) {
	got := todayWIB()
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Fatalf("todayWIB = %s, want midnight", got.Format("15:04:05.999999999"))
	}
	// Its date must be the current WIB date, even when the host clock is UTC.
	wib := store.WallClock(time.Now())
	if got.Format("2006-01-02") != wib.Format("2006-01-02") {
		t.Fatalf("todayWIB date = %s, want %s", got.Format("2006-01-02"), wib.Format("2006-01-02"))
	}
}

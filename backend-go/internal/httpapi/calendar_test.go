package httpapi

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestCalendarSpanFallsBackAndCrossesMidnight(t *testing.T) {
	date := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)

	// A package with no session length still gets two hours, not zero.
	start, end, endDate := calendarSpan(date, "05:30", 0)
	if start != "053000" || end != "073000" || !endDate.Equal(date) {
		t.Fatalf("fallback span = %s-%s on %s", start, end, endDate.Format("2006-01-02"))
	}

	// A late session ends after midnight, on the next day.
	start, end, endDate = calendarSpan(date, "23:30", 120)
	if start != "233000" || end != "013000" || !endDate.Equal(date.AddDate(0, 0, 1)) {
		t.Fatalf("midnight span = %s-%s on %s", start, end, endDate.Format("2006-01-02"))
	}
}

func TestCalendarStatus(t *testing.T) {
	// The photographer's accepted sessions are confirmed; anything still open is
	// tentative, so the calendar shows it without claiming the slot.
	for status, want := range map[string]string{
		"ACCEPTED":             "CONFIRMED",
		"COMPLETED":            "CONFIRMED",
		"PENDING":              "TENTATIVE",
		"RESCHEDULE_REQUESTED": "TENTATIVE",
	} {
		if got := calendarStatus(status); got != want {
			t.Errorf("calendarStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestEscapeICSText(t *testing.T) {
	// RFC 5545 gives these characters meaning, so client input must not reach the
	// feed raw: a comma in a name would split the field.
	got := escapeICSText("a,b;c\\d\ne")
	want := `a\,b\;c\\d\ne`
	if got != want {
		t.Fatalf("escapeICSText = %q, want %q", got, want)
	}
}

func TestFoldICSLineKeepsRunesWhole(t *testing.T) {
	line := "SUMMARY:" + strings.Repeat("é", 60) // 8 + 120 bytes
	folded := foldICSLine(line)
	for _, part := range strings.Split(folded, "\r\n") {
		if len(part) > 75 {
			t.Fatalf("line is %d octets: %q", len(part), part)
		}
		if !utf8.ValidString(part) {
			t.Fatalf("fold split a character: %q", part)
		}
	}
	// Unfolding must give back exactly what went in.
	if got := strings.ReplaceAll(folded, "\r\n ", ""); got != line {
		t.Fatalf("unfolded = %q, want %q", got, line)
	}
}

func TestBuildCalendarRendersBookings(t *testing.T) {
	stamp := time.Date(2026, 11, 26, 4, 0, 0, 0, time.UTC)
	events := []calendarEvent{
		{
			ID: 12, Name: "Chelsea, Putri", Package: "Emerge", Location: "Balairung; UGM",
			Date: time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC), Time: "05:30",
			Status: "ACCEPTED", Minutes: 0,
		},
		{
			ID: 13, Name: "Della Fitriani", Package: "Origin",
			Date: time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC), Time: "23:30",
			Status: "PENDING", Minutes: 120,
		},
		{
			ID: 14, Name: "Tanpa Jam",
			Date:   time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC),
			Status: "COMPLETED",
		},
	}
	feed := buildCalendar(events, stamp)

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n",
		"X-WR-TIMEZONE:Asia/Jakarta",
		"UID:booking-12@elvove.site",
		"DTSTAMP:20261126T040000Z",
		"DTSTART;TZID=Asia/Jakarta:20261125T053000",
		`DTEND;TZID=Asia/Jakarta:20261125T073000`, // fallback of two hours
		`SUMMARY:Chelsea\, Putri · Emerge`,
		`LOCATION:Balairung\; UGM`,
		"STATUS:CONFIRMED",
		"DTSTART;TZID=Asia/Jakarta:20261125T233000",
		"DTEND;TZID=Asia/Jakarta:20261126T013000", // crosses midnight
		"STATUS:TENTATIVE",
		"DTSTART;VALUE=DATE:20261127",
		"DTEND;VALUE=DATE:20261128",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(feed, want) {
			t.Errorf("feed is missing %q", want)
		}
	}
	// Every break must be CRLF: a bare LF is not valid iCalendar.
	if strings.Contains(strings.ReplaceAll(feed, "\r\n", ""), "\n") {
		t.Error("feed contains a bare LF")
	}
	for _, line := range strings.Split(feed, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line is %d octets: %q", len(line), line)
		}
	}
}

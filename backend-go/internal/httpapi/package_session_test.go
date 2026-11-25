package httpapi

import "testing"

// A package's session length feeds the calendar's end time, so the bounds are
// worth pinning: zero means "not set yet" (the calendar then assumes two
// hours), and anything past a day is a typo rather than a real booking.
func TestSessionMinutesBounds(t *testing.T) {
	for _, minutes := range []int{0, 1, 120, 1440} {
		if message := sessionMinutesError(minutes); message != "" {
			t.Fatalf("sessionMinutesError(%d) = %q, want it accepted", minutes, message)
		}
	}
	for _, minutes := range []int{-1, 1441, 100000} {
		if message := sessionMinutesError(minutes); message == "" {
			t.Fatalf("sessionMinutesError(%d) accepted, want it rejected", minutes)
		}
	}
}

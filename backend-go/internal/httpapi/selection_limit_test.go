package httpapi

import (
	"strings"
	"testing"
)

// The quota rule is one expression shared by the client page, the e-mail and
// the admin panel. These checks pin the SQL text so a change in one place is
// impossible without failing here: a snapshot of 0 falls back to the package's
// quota, and a package without a quota answers 0.
func TestSelectionLimitSQLFallsBackToPackage(t *testing.T) {
	const want = "COALESCE(NULLIF(b.selection_limit,0), (SELECT p.photo_selection_limit FROM packages p WHERE p.title = b.package_title ORDER BY p.id LIMIT 1), 0)"
	if got := selectionLimitSQL("b"); got != want {
		t.Errorf("selectionLimitSQL(b) =\n%s\nwant\n%s", got, want)
	}
	if got := selectionLimitSQL("bookings"); got == selectionLimitSQL("b") {
		t.Error("the alias must be substituted, not fixed")
	}
}

// The admin panel must report the same quota the client page and the e-mails
// use, otherwise a booking made while packages offered none shows 0 while the
// client may pick its package's quota.
func TestBookingColumnsUseTheSharedSelectionLimit(t *testing.T) {
	if !strings.Contains(bookingBaseColumns, selectionLimitSQL("bookings")) {
		t.Error("bookingBaseColumns does not use selectionLimitSQL")
	}
	if strings.Contains(bookingBaseColumns, "COALESCE(selection_limit, 0)") {
		t.Error("bookingBaseColumns still reads the raw snapshot")
	}
}

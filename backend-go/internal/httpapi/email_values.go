package httpapi

// The invoice and selection values the templates show: numbers, dates, amounts
// and titles derived from a booking, plus the escaping every interpolated value
// goes through.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// emailInvoiceNumber derives the number from the booking itself, so the same
// booking always gets the same number without a counter table:
// INV-<year>-<id>, the id zero-padded to four digits.
func emailInvoiceNumber(id int64, year int) string {
	digits := strconv.FormatInt(id, 10)
	for len(digits) < 4 {
		digits = "0" + digits
	}
	return "INV-" + strconv.Itoa(year) + "-" + digits
}

// emailDueDate is the day before the session: the point by which the balance
// should be settled.
func emailDueDate(eventDate string) string {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(eventDate))
	if err != nil {
		return emailTanggal(eventDate)
	}
	return emailTanggal(date.AddDate(0, 0, -1).Format("2006-01-02"))
}

// emailSettlementDeadline words the payment point for the day the notice goes
// out: before the session it is the day before that session, and once the
// session has passed (the notice is sent after the shoot, together with the
// photo selection) a date in the past would be wrong.
func emailSettlementDeadline(eventDate, issuedAt string) string {
	due := emailDueDate(eventDate)
	session, err := time.Parse("2006-01-02", strings.TrimSpace(eventDate))
	issued, issuedErr := time.Parse("2006-01-02", strings.TrimSpace(issuedAt))
	if err != nil || issuedErr != nil || !issued.After(session) {
		return "paling lambat " + due
	}
	return "secepatnya"
}

// emailSelectionNote is the one line under the photo-selection button.
// A package whose quota is still zero gets a link that only shows the photos.
// Promising a choice the page refuses would read as a broken e-mail.
func emailSelectionTitle(limit int) string {
	if limit < 1 {
		return "FOTO SESI ANDA"
	}
	return "PILIH FOTO UNTUK DIEDIT"
}

func emailSelectionButton(limit int) string {
	if limit < 1 {
		return "LIHAT FOTO"
	}
	return "PILIH FOTO SEKARANG"
}

func emailSelectionNote(limit int) string {
	if limit < 1 {
		return "Foto sesi Anda sudah siap dilihat. Hubungi kami bila ingin memilih foto untuk diedit."
	}
	return fmt.Sprintf("Pilih maksimal %d foto yang ingin Anda minta untuk diedit.", limit)
}

// emailSiteURL mirrors Java's constructor: a blank address falls back to the
// production site so the logo and footer links never break.
func emailSiteURL(siteURL string) string {
	siteURL = strings.TrimSpace(siteURL)
	if siteURL == "" {
		return emailSiteFallback
	}
	return siteURL
}

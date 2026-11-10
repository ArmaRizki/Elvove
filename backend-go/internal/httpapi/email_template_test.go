package httpapi

import (
	"strings"
	"testing"
)

func sampleBooking() bookingEmail {
	return bookingEmail{
		Name:          "Budi Santoso",
		EventDate:     "2026-11-09",
		EventTime:     "09:30",
		EventLocation: "Jakarta Selatan",
		PackageTitle:  "Premium",
		PackagePrice:  "2000000.00",
	}
}

func TestRenderBookingConfirmationMatchesJavaLayout(t *testing.T) {
	subject, html, text := renderBookingConfirmation(sampleBooking(), "Elvove", "elvo.vee@gmail.com", "https://elvove.site")

	if subject != "Booking Anda Telah Dikonfirmasi — Elvove" {
		t.Errorf("subject = %q", subject)
	}

	// The document skeleton Java's BookingEmailTemplate.document() produces.
	for _, marker := range []string{
		"<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\"",
		`xmlns:o="urn:schemas-microsoft-com:office:office" lang="id"`,
		`<meta charset="UTF-8">`,
		`<title>Konfirmasi Booking — Elvove</title>`,
		`<!--[if mso]><table role="presentation" width="600"`,
		`</body></html>`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing document marker %q", marker)
		}
	}

	// Header, card, footer and the interactive blocks.
	for _, marker := range []string{
		`src="https://elvove.site/logo.png"`,
		`>ELVOVE</td>`,
		`PESANAN DITERIMA`,
		`Booking Anda Telah Dikonfirmasi`,
		`DETAIL BOOKING`,
		`LANGKAH SELANJUTNYA`,
		`BALAS EMAIL KAMI`,
		`mailto:elvo.vee@gmail.com`,
		`Capturing Moments, Creating Memories.`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing marker %q", marker)
		}
	}

	// Booking values, formatted the way the Java template formats them.
	for _, marker := range []string{
		"Budi Santoso",
		"9 November 2026",
		"09:30",
		"Jakarta Selatan",
		"Premium",
		"Rp2.000.000",
		"Akan dikonfirmasi",
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing booking value %q", marker)
		}
	}

	// The plain-text alternative mirrors the same content.
	for _, marker := range []string{
		"PESANAN DITERIMA",
		"DETAIL BOOKING",
		"Nama Pemesan    : Budi Santoso",
		"Tanggal Booking : 9 November 2026",
		"Total Harga     : Rp2.000.000",
		"Sisa Pembayaran : Akan dikonfirmasi",
		"LANGKAH SELANJUTNYA",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing marker %q", marker)
		}
	}

	// The hidden preview line shown in the inbox list.
	if !strings.Contains(html, "Booking Anda Telah Dikonfirmasi — Budi Santoso. Sesi foto Anda pada 9 November 2026.") {
		t.Error("html missing inbox preview line")
	}
}

func TestRenderBookingConfirmationEscapesBookingValues(t *testing.T) {
	booking := sampleBooking()
	booking.Name = `<script>alert("x")</script>`
	booking.EventLocation = `Kantor & "Studio"`
	_, html, text := renderBookingConfirmation(booking, "", "", "")

	if strings.Contains(html, "<script>") {
		t.Error("html contains unescaped markup from the booking name")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("html does not escape the booking name")
	}
	if !strings.Contains(html, "Kantor &amp; &quot;Studio&quot;") {
		t.Error("html does not escape the location")
	}
	// The plain-text part keeps the raw value, as Java's does.
	if !strings.Contains(text, `<script>alert("x")</script>`) {
		t.Error("text should keep the raw booking name")
	}
}

func TestRenderBookingConfirmationUsesBrandFallbacks(t *testing.T) {
	subject, html, text := renderBookingConfirmation(sampleBooking(), "  ", "", "")

	if subject != "Booking Anda Telah Dikonfirmasi — Elvove" {
		t.Errorf("subject = %q", subject)
	}
	// No sender address: the reply button falls back to the site URL instead
	// of a broken "mailto:" with no address.
	if strings.Contains(html, "mailto:") {
		t.Error("html should not build a mailto: link without a sender address")
	}
	if !strings.Contains(html, `href="https://elvove.site"`) {
		t.Error("html should fall back to the site URL for the reply button")
	}
	if !strings.Contains(text, "Terima kasih telah memilih Elvove.") {
		t.Error("text should fall back to the brand name")
	}
}

func TestRenderBookingConfirmationMissingValuesUsePendingMarker(t *testing.T) {
	booking := bookingEmail{Name: "Andi"}
	_, html, text := renderBookingConfirmation(booking, "Elvove", "a@b.c", "")

	// The HTML card renders each pending value in its own table cell, so only
	// the marker itself is asserted there: five card rows (date, time, total,
	// down payment, balance) plus the hidden inbox preview line.
	if count := strings.Count(html, emailPending); count != 6 {
		t.Errorf("html pending markers = %d, want 6", count)
	}
	for _, label := range []string{"Tanggal Booking", "Waktu Sesi", "Total Harga"} {
		if !strings.Contains(html, ">"+label+"</td>") {
			t.Errorf("html missing detail row %q", label)
		}
	}
	for _, marker := range []string{
		"Tanggal Booking : Akan dikonfirmasi",
		"Waktu Sesi      : Akan dikonfirmasi",
		"Total Harga     : Akan dikonfirmasi",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing pending marker %q", marker)
		}
	}
}

func TestEmailHelpers(t *testing.T) {
	if got := emailRupiah("2000000.00"); got != "Rp2.000.000" {
		t.Errorf("emailRupiah = %q", got)
	}
	if got := emailRupiah("999"); got != "Rp999" {
		t.Errorf("emailRupiah = %q", got)
	}
	if got := emailRupiah(""); got != emailPending {
		t.Errorf("emailRupiah = %q", got)
	}
	if got := emailTanggal("2026-01-05"); got != "5 Januari 2026" {
		t.Errorf("emailTanggal = %q", got)
	}
	if got := emailJam("14:05:07"); got != "14:05" {
		t.Errorf("emailJam = %q", got)
	}
	if got := emailOneLine(" A\nB\tC "); got != "A B C" {
		t.Errorf("emailOneLine = %q", got)
	}
	if got := emailLogoURL("https://elvove.site/"); got != "https://elvove.site/logo.png" {
		t.Errorf("emailLogoURL = %q", got)
	}
}

func TestMimeWordEncodesNonASCIISubject(t *testing.T) {
	if got := mimeWord("Plain subject"); got != "Plain subject" {
		t.Errorf("mimeWord = %q", got)
	}
	got := mimeWord("Booking Anda Telah Dikonfirmasi — Elvove")
	if !strings.HasPrefix(got, "=?UTF-8?B?") || !strings.HasSuffix(got, "?=") {
		t.Errorf("mimeWord = %q", got)
	}
	// net/mail quotes the display name; the result is a valid address either
	// way, so only the shape is asserted.
	if from := formatMailFrom("Elvove", "a@b.c"); !strings.Contains(from, "<a@b.c>") || !strings.Contains(from, "Elvove") {
		t.Errorf("formatMailFrom = %q", from)
	}
	if from := formatMailFrom("  ", "a@b.c"); from != "a@b.c" {
		t.Errorf("formatMailFrom = %q", from)
	}
	if from := formatMailFrom("Fotografer Elvove", "a@b.c"); !strings.Contains(from, "<a@b.c>") {
		t.Errorf("formatMailFrom = %q", from)
	}
}

func TestRenderBookingRescheduleMatchesJavaLayout(t *testing.T) {
	subject, html, text := renderBookingReschedule(sampleBooking(), "Elvove", "elvo.vee@gmail.com", "https://elvove.site")

	if subject != "Jadwal Sesi Foto Diperbarui — Elvove" {
		t.Errorf("subject = %q", subject)
	}
	for _, marker := range []string{
		`PERUBAHAN JADWAL`,
		`Jadwal Sesi Foto Diperbarui`,
		`Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.`,
		`DETAIL BOOKING`,
		`LANGKAH SELANJUTNYA`,
		`mailto:elvo.vee@gmail.com`,
		`Rp2.000.000`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing marker %q", marker)
		}
	}
	for _, marker := range []string{
		"PERUBAHAN JADWAL\n\n",
		"Jadwal Sesi Foto Diperbarui\n\n",
		"Kami ingin segera menginformasikan usulan perubahan jadwal sesi foto Anda.",
		"Nama Pemesan    : Budi Santoso",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing marker %q", marker)
		}
	}
}

func TestRenderAdminNotificationMatchesJavaLayout(t *testing.T) {
	booking := sampleBooking()
	booking.ID = 42
	booking.Email = "budi@example.com"
	booking.Phone = "08123456789"
	booking.Status = "ACCEPTED"
	link := "https://elvove.site/admin/bookings?booking=42"

	subject, html, text := renderAdminNotification(booking, "Elvove", link, "https://elvove.site")

	if subject != "Booking Baru — Budi Santoso" {
		t.Errorf("subject = %q", subject)
	}
	for _, marker := range []string{
		`BOOKING BARU`,
		`Ada Pemesanan Baru`,
		`>Pemesanan baru dari Budi Santoso.</td>`,
		`>budi@example.com</td>`,
		`>08123456789</td>`,
		`>Diterima</td>`,
		`BUKA DI PANEL ADMIN`,
		`href="` + link + `"`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing marker %q", marker)
		}
	}
	for _, marker := range []string{
		"BOOKING BARU\n\n",
		"Pemesanan baru dari Budi Santoso.",
		"Email           : budi@example.com",
		"Status          : Diterima",
		"Buka panel admin: " + link,
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing marker %q", marker)
		}
	}
}

func TestAdminBookingURL(t *testing.T) {
	got, err := adminBookingURL("https://elvove.site/", 7)
	if err != nil || got != "https://elvove.site/admin/bookings?booking=7" {
		t.Fatalf("adminBookingURL = %q, %v", got, err)
	}
	if _, err := adminBookingURL("http://localhost:5173", 7); err != nil {
		t.Errorf("local admin URL rejected: %v", err)
	}
	for _, base := range []string{"", "  ", "ftp://elvove.site", "https://user:pass@elvove.site", "https://elvove.site/?a=1", "https://elvove.site/#x", "not a url"} {
		if _, err := adminBookingURL(base, 7); err == nil {
			t.Errorf("adminBookingURL(%q) should be rejected", base)
		}
	}
}

func TestDefaultTermsContentMatchesJava(t *testing.T) {
	// The two sentences Java's TermsService.DEFAULT_CONTENT has and an older Go
	// seed lacked, plus the trailing newline the Java text block keeps.
	for _, marker := range []string{
		"- A down payment is required to secure a confirmed photography session.\n",
		"- The down payment is non-refundable if the client cancels the session unilaterally.\n",
		"- ELVOVE will respect any publication restrictions expressly agreed upon with the client.\n",
	} {
		if !strings.Contains(defaultTermsContent, marker) {
			t.Errorf("defaultTermsContent missing %q", marker)
		}
	}
	if !strings.HasSuffix(defaultTermsContent, "by the relevant parties.\n") {
		t.Error("defaultTermsContent must end with a newline, like the Java text block")
	}
}

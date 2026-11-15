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
	subject, html, text := renderBookingConfirmation(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")

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
		`src="cid:elvove-logo"`,
		`>ELVOVE</td>`,
		`PESANAN DITERIMA`,
		`Booking Anda Telah Dikonfirmasi`,
		`DETAIL BOOKING`,
		`LANGKAH SELANJUTNYA`,
		`CHAT VIA WHATSAPP`,
		`https://wa.me/6281234567890`,
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
		// Half the package price is the DP; the balance is the other half.
		"Rp1.000.000",
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing booking value %q", marker)
		}
	}

	// The payment card: bank transfer details and the hosted QRIS image.
	for _, marker := range []string{
		`PEMBAYARAN`,
		`DP 50% DIBAYAR SEKARANG`,
		`Transfer BCA`,
		`>8025325911</td>`,
		`a.n. Leonardus Seco Patu Sila Anggara`,
		`src="cid:elvove-qris"`,
		`atau scan QRIS`,
		`ELVOVE - ART &middot; NMID ID1026639151079`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing payment marker %q", marker)
		}
	}

	// The plain-text alternative mirrors the same content.
	for _, marker := range []string{
		"PESANAN DITERIMA",
		"DETAIL BOOKING",
		"Nama Pemesan    : Budi Santoso",
		"Tanggal Booking : 9 November 2026",
		"Total Harga     : Rp2.000.000",
		"DP 50%          : Rp1.000.000",
		"Sisa Pembayaran : Rp1.000.000",
		"PEMBAYARAN",
		"DP 50%          : Rp1.000.000 (dibayar sekarang)",
		"Transfer BCA    : 8025325911",
		"Atas Nama       : Leonardus Seco Patu Sila Anggara",
		"QRIS            : https://elvove.site/qris.png",
		"LANGKAH SELANJUTNYA",
		"WhatsApp: https://wa.me/6281234567890",
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

// The customer and admin e-mails are black ink on white: the only colour left
// is the wordmark in the header and the WhatsApp button. Anything that
// reintroduces a brand-red accent outside the wordmark fails here.
func TestEmailsKeepOnlyTheWordmarkAndWhatsAppInColour(t *testing.T) {
	admin := sampleBooking()
	admin.ID = 42
	admin.Email = "budi@example.com"
	admin.Status = "ACCEPTED"

	for name, html := range map[string]string{
		"confirmation": func() string {
			_, h, _ := renderBookingConfirmation(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")
			return h
		}(),
		"reschedule": func() string {
			_, h, _ := renderBookingReschedule(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")
			return h
		}(),
		"admin": func() string {
			_, h, _ := renderAdminNotification(admin, "Elvove", "https://elvove.site/admin/bookings?booking=42", "https://elvove.site")
			return h
		}(),
		"invoice": func() string {
			_, h, _ := renderBookingInvoice(sampleBooking(), invoiceDetails{Number: "INV-2026-0012", IssuedAt: "2026-11-16"}, "Elvove", "081234567890", "https://elvove.site")
			return h
		}(),
		"settlement": func() string {
			_, h, _ := renderBookingSettlement(sampleBooking(), invoiceDetails{Number: "INV-2026-0012", IssuedAt: "2026-11-16", DownPaidAt: "2026-11-10"}, "Elvove", "081234567890", "https://elvove.site")
			return h
		}(),
	} {
		if count := strings.Count(html, emailBrandRed); count != 1 {
			t.Errorf("%s: brand red appears %d times, want only the wordmark", name, count)
		}
		if !strings.Contains(html, "color:"+emailBrandRed+";\">ELVOVE</td>") {
			t.Errorf("%s: the one brand-red use is not the ELVOVE wordmark", name)
		}
		for _, retired := range []string{"#F4F1F8", "#FCEEF1", "#F8F7FB", "#EEEAF2", "#858191", "#292638"} {
			if strings.Contains(html, retired) {
				t.Errorf("%s: retired colour %s is still in the e-mail", name, retired)
			}
		}
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
	// No WhatsApp number: the button falls back to the site URL instead of a
	// dead wa.me link with no number.
	if strings.Contains(html, "wa.me") {
		t.Error("html should not build a wa.me link without a WhatsApp number")
	}
	if !strings.Contains(html, `href="https://elvove.site"`) {
		t.Error("html should fall back to the site URL for the WhatsApp button")
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
	// down payment, balance), the amount due in the payment card, plus the
	// hidden inbox preview line.
	if count := strings.Count(html, emailPending); count != 7 {
		t.Errorf("html pending markers = %d, want 7", count)
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
	if got := emailDownPayment("2000000.00"); got != "Rp1.000.000" {
		t.Errorf("emailDownPayment = %q", got)
	}
	// An odd price keeps the two halves adding up to the total.
	if got := emailDownPayment("1000001"); got != "Rp500.000" {
		t.Errorf("emailDownPayment odd = %q", got)
	}
	if got := emailRemainingBalance("1000001"); got != "Rp500.001" {
		t.Errorf("emailRemainingBalance odd = %q", got)
	}
	if got := emailDownPayment(""); got != emailPending {
		t.Errorf("emailDownPayment empty = %q", got)
	}
	if got := emailRemainingBalance("bukan angka"); got != emailPending {
		t.Errorf("emailRemainingBalance invalid = %q", got)
	}
	if got := emailQRISURL("https://elvove.site/"); got != "https://elvove.site/qris.png" {
		t.Errorf("emailQRISURL = %q", got)
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

func TestEmailWhatsAppURLNormalisesTheStoredNumber(t *testing.T) {
	cases := map[string]string{
		"081234567890":     "https://wa.me/6281234567890",
		"+62 812-3456-789": "https://wa.me/628123456789",
		"6281234567890":    "https://wa.me/6281234567890",
		"":                 "",
		"   ":              "",
	}
	for number, want := range cases {
		if got := emailWhatsAppURL(number); got != want {
			t.Errorf("emailWhatsAppURL(%q) = %q, want %q", number, got, want)
		}
	}
}

func TestRenderBookingRescheduleMatchesJavaLayout(t *testing.T) {
	subject, html, text := renderBookingReschedule(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")

	if subject != "Jadwal Sesi Foto Diperbarui — Elvove" {
		t.Errorf("subject = %q", subject)
	}
	for _, marker := range []string{
		`PERUBAHAN JADWAL`,
		`Jadwal Sesi Foto Diperbarui`,
		`Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.`,
		`DETAIL BOOKING`,
		`LANGKAH SELANJUTNYA`,
		`https://wa.me/6281234567890`,
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

// The invoice is the last message of a booking: the transaction as it was paid
// and the Drive link with the edited photos. It asks for no money, so it must
// not carry the account or the QRIS code.
func TestRenderBookingInvoice(t *testing.T) {
	invoice := invoiceDetails{
		Number:      emailInvoiceNumber(12, 2026),
		IssuedAt:    "2026-11-16",
		DownPaidAt:  "2026-11-10",
		SettledAt:   "2026-11-15",
		DeliveryURL: "https://drive.google.com/drive/folders/abc123",
	}
	subject, html, text := renderBookingInvoice(sampleBooking(), invoice, "Elvove", "081234567890", "https://elvove.site")

	if subject != "Invoice INV-2026-0012 — Elvove" {
		t.Errorf("subject = %q", subject)
	}
	for _, marker := range []string{
		`>INVOICE</td>`,
		`Invoice &amp; Hasil Foto`,
		`Nomor INV-2026-0012`,
		`Lunas pada 15 November 2026`,
		`RINCIAN TRANSAKSI`,
		`DP 50% Dibayar`,
		`Pelunasan Dibayar`,
		`FOTO SIAP DIUNDUH`,
		`https://drive.google.com/drive/folders/abc123`,
		`BUKA GALERI FOTO`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing marker %q", marker)
		}
	}
	for _, absent := range []string{`>8025325911</td>`, `src="cid:elvove-qris"`, `CARA MELUNASI`} {
		if strings.Contains(html, absent) {
			t.Errorf("a closed invoice must not ask for payment, found %q", absent)
		}
	}
	for _, marker := range []string{
		"INVOICE\n\n",
		"Invoice & Hasil Foto",
		"Nomor INV-2026-0012 · diterbitkan 16 November 2026.",
		"Nomor Invoice    : INV-2026-0012",
		"Status           : Lunas pada 15 November 2026",
		"DP 50% Dibayar   : Rp1.000.000",
		"Pelunasan Dibayar: Rp1.000.000",
		"FOTO SIAP DIUNDUH",
		"Foto yang sudah diedit: https://drive.google.com/drive/folders/abc123",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing marker %q", marker)
		}
	}
	for _, absent := range []string{"Transfer BCA", "QRIS"} {
		if strings.Contains(text, absent) {
			t.Errorf("a closed invoice must not ask for payment, found %q", absent)
		}
	}
}

// An unrecorded payment still renders, so the admin can resend an invoice while
// the record is being fixed up rather than being blocked by the template.
func TestRenderBookingInvoiceWithoutRecordedPayments(t *testing.T) {
	invoice := invoiceDetails{Number: "INV-2026-0012", IssuedAt: "2026-11-16", DeliveryURL: "https://drive.google.com/x"}
	_, html, _ := renderBookingInvoice(sampleBooking(), invoice, "Elvove", "", "https://elvove.site")

	if !strings.Contains(html, "Belum ada pembayaran") {
		t.Error("html should state that no payment is recorded")
	}
	if !strings.Contains(html, "https://drive.google.com/x") {
		t.Error("html should still carry the delivery link")
	}
}

func TestRenderBookingSettlement(t *testing.T) {
	invoice := invoiceDetails{Number: "INV-2026-0012", IssuedAt: "2026-11-16", DownPaidAt: "2026-11-10"}
	subject, html, text := renderBookingSettlement(sampleBooking(), invoice, "Elvove", "081234567890", "https://elvove.site")

	if subject != "Pelunasan Sisa Pembayaran — Elvove" {
		t.Errorf("subject = %q", subject)
	}
	for _, marker := range []string{
		`>PELUNASAN</td>`,
		`Pelunasan Sisa Pembayaran`,
		`RINCIAN PELUNASAN`,
		`SISA PEMBAYARAN`,
		`Rp1.000.000`,
		// The balance is due the day before the session.
		`8 November 2026`,
		`CARA MELUNASI`,
		`>8025325911</td>`,
		`src="cid:elvove-qris"`,
	} {
		if !strings.Contains(html, marker) {
			t.Errorf("html missing marker %q", marker)
		}
	}
	for _, marker := range []string{
		"PELUNASAN\n\n",
		"Sisa Pembayaran : Rp1.000.000",
		"paling lambat 8 November 2026",
		"Kirim bukti transfer via WhatsApp",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("text missing marker %q", marker)
		}
	}
}

func TestInvoiceNumberAndDueDate(t *testing.T) {
	if got := emailInvoiceNumber(12, 2026); got != "INV-2026-0012" {
		t.Errorf("emailInvoiceNumber = %q", got)
	}
	if got := emailInvoiceNumber(12345, 2026); got != "INV-2026-12345" {
		t.Errorf("emailInvoiceNumber long id = %q", got)
	}
	if got := emailDueDate("2026-11-09"); got != "8 November 2026" {
		t.Errorf("emailDueDate = %q", got)
	}
	if got := emailDueDate(""); got != emailPending {
		t.Errorf("emailDueDate empty = %q", got)
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

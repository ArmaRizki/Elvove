package httpapi

// Booking e-mail templates. This is a direct port of the Java
// BookingEmailTemplate: table-based markup with inline CSS only, so Gmail,
// Outlook, Apple Mail and mobile clients render the same layout. Every value
// that comes from a booking is escaped before it is interpolated, so customer
// input can never inject markup into the message.

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// Inter from Google Fonts where the client loads web fonts; Arial and
	// Helvetica are used everywhere else, so the text stays readable.
	emailFont       = "Inter, Arial, Helvetica, sans-serif"
	emailFontImport = "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"

	// Palette: black ink on white. Only the logo, the ELVOVE wordmark and the
	// WhatsApp button keep a colour, so the message reads as one monochrome
	// document.
	emailPage     = "#F5F5F5"
	emailCard     = "#FFFFFF"
	emailInk      = "#000000"
	emailOnInk    = "#FFFFFF"
	emailBrandRed = "#E11D48" // the ELVOVE wordmark in the header, nothing else
	emailInfoBg   = "#FFFFFF"
	emailBorder   = "#E5E5E5"
	emailBadgeBg  = "#000000"
	// The same green as the floating WhatsApp button on the site.
	emailWhatsAppGreen = "#25D366"

	emailBrandFallback = "Elvove"
	emailTagline       = "Capturing Moments, Creating Memories."
	emailPending       = "Akan dikonfirmasi"
	emailSiteFallback  = "https://elvove.site"
)

var emailLineBreaks = regexp.MustCompile(`[\r\n\t]+`)
var emailNonDigits = regexp.MustCompile(`\D`)

var emailMonths = []string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// bookingEmail holds the booking values the customer-facing templates render.
type bookingEmail struct {
	ID            int64
	Name          string
	Email         string
	Phone         string
	EventDate     string // YYYY-MM-DD; empty when unknown
	EventTime     string // HH:MM or HH:MM:SS; empty when unknown
	EventLocation string
	PackageTitle  string
	PackagePrice  string // decimal string; empty when unknown
	Status        string // BookingStatus constant; empty when unknown
}

// renderBookingConfirmation returns the subject, HTML body and plain-text body
// of the booking confirmation, matching the Java template byte for byte in
// wording and layout.
func renderBookingConfirmation(b bookingEmail, senderName, whatsapp, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)

	body := emailBadge("PESANAN DITERIMA") +
		emailHeadingBlock(
			"Booking Anda Telah Dikonfirmasi",
			"Detail reservasi Anda sudah tercatat. Tim kami akan menghubungi Anda untuk konfirmasi lebih lanjut.") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". Kami telah menerima dan mengonfirmasi booking Anda. "+
				"Berikut adalah detail reservasi Anda.") +
		emailDetailsCard(b) +
		emailPaymentBlock(b, siteURL) +
		emailNextStepsBlock() +
		emailWhatsAppButton(whatsapp, siteURL)

	var text strings.Builder
	text.WriteString("PESANAN DITERIMA\n\n")
	text.WriteString("Booking Anda Telah Dikonfirmasi\n\n")
	text.WriteString("Detail reservasi Anda sudah tercatat. Tim kami akan menghubungi Anda untuk konfirmasi lebih lanjut.\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". ")
	text.WriteString("Kami telah menerima dan mengonfirmasi booking Anda. ")
	text.WriteString("Berikut adalah detail reservasi Anda.\n\n")
	text.WriteString("DETAIL BOOKING\n")
	text.WriteString(emailDetailLines(b))
	text.WriteString("\n" + emailPaymentLines(b.PackagePrice, siteURL))
	text.WriteString("\nLANGKAH SELANJUTNYA\n")
	text.WriteString(emailStepsText())
	if url := emailWhatsAppURL(whatsapp); url != "" {
		text.WriteString("\nWhatsApp: " + url + "\n")
	}
	text.WriteString("\nElvove\n")
	text.WriteString(emailTagline + "\n")
	text.WriteString("© " + strconv.Itoa(time.Now().Year()) + " " + brand + ". All rights reserved.\n")
	text.WriteString(siteURL + "\n")

	subject := "Booking Anda Telah Dikonfirmasi — " + brand
	preview := emailPreview(b, "Booking Anda Telah Dikonfirmasi")
	return subject, emailDocument(body, preview, siteURL), text.String()
}

// renderBookingReschedule is the schedule-change notice. It shares the
// confirmation layout so the customer always sees one consistent design.
func renderBookingReschedule(b bookingEmail, senderName, whatsapp, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)

	body := emailBadge("PERUBAHAN JADWAL") +
		emailHeadingBlock(
			"Jadwal Sesi Foto Diperbarui",
			"Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". Kami ingin segera menginformasikan usulan perubahan "+
				"jadwal sesi foto Anda. Hubungi kami via WhatsApp untuk mengonfirmasi atau berdiskusi.") +
		emailDetailsCard(b) +
		emailNextStepsBlock() +
		emailWhatsAppButton(whatsapp, siteURL)

	var text strings.Builder
	text.WriteString("PERUBAHAN JADWAL\n\n")
	text.WriteString("Jadwal Sesi Foto Diperbarui\n\n")
	text.WriteString("Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". ")
	text.WriteString("Kami ingin segera menginformasikan usulan perubahan jadwal sesi foto Anda. ")
	text.WriteString("Mohon hubungi kami via WhatsApp untuk mengonfirmasi atau berdiskusi.\n\n")
	text.WriteString("DETAIL BOOKING\n")
	text.WriteString(emailDetailLines(b))
	text.WriteString("\nLANGKAH SELANJUTNYA\n")
	text.WriteString(emailStepsText())
	if url := emailWhatsAppURL(whatsapp); url != "" {
		text.WriteString("\nWhatsApp: " + url + "\n")
	}
	text.WriteString("\nElvove\n")
	text.WriteString(emailTagline + "\n")
	text.WriteString("© " + strconv.Itoa(time.Now().Year()) + " " + brand + ". All rights reserved.\n")
	text.WriteString(siteURL + "\n")

	subject := "Jadwal Sesi Foto Diperbarui — " + brand
	preview := emailPreview(b, "Jadwal Sesi Foto Diperbarui")
	return subject, emailDocument(body, preview, siteURL), text.String()
}

// renderAdminNotification is the administrator notification for a new booking:
// the fields the team needs plus a link to the booking in the admin panel.
func renderAdminNotification(b bookingEmail, senderName, link, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)

	body := emailBadge("BOOKING BARU") +
		emailHeadingBlock(
			"Ada Pemesanan Baru",
			"Sebuah pemesanan baru telah tersimpan di sistem "+brand+".") +
		emailGreetingBlock(
			"Pemesanan baru dari "+customer+".",
			"Berikut detail pemesanan yang baru masuk. Buka panel admin untuk meninjau dan mengubah status.") +
		emailAdminDetailsCard(b) +
		emailAdminButton(link)

	var text strings.Builder
	text.WriteString("BOOKING BARU\n\n")
	text.WriteString("Ada Pemesanan Baru\n\n")
	text.WriteString("Pemesanan baru dari " + customer + ".\n\n")
	text.WriteString("DETAIL BOOKING\n")
	text.WriteString(emailAdminDetailLines(b))
	text.WriteString("\nBuka panel admin: " + link + "\n\n")
	text.WriteString("Elvove\n")
	text.WriteString(emailTagline + "\n")

	subject := "Booking Baru — " + customer
	preview := "Pemesanan baru dari " + customer + "."
	return subject, emailDocument(body, preview, siteURL), text.String()
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

// ---------------------------------------------------------------- layout

func emailDocument(body, preheader, siteURL string) string {
	return "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" " +
		"\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">" +
		"<html xmlns=\"http://www.w3.org/1999/xhtml\" " +
		"xmlns:v=\"urn:schemas-microsoft-com:vml\" " +
		"xmlns:o=\"urn:schemas-microsoft-com:office:office\" lang=\"id\">" +
		"<head>" +
		"<meta charset=\"UTF-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">" +
		"<meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge\">" +
		"<meta name=\"color-scheme\" content=\"light\">" +
		"<meta name=\"supported-color-schemes\" content=\"light\">" +
		"<link rel=\"preconnect\" href=\"https://fonts.googleapis.com\">" +
		"<link href=\"" + emailFontImport + "\" rel=\"stylesheet\">" +
		"<title>Konfirmasi Booking — Elvove</title>" +
		"<!--[if mso]><noscript><xml><o:OfficeDocumentSettings>" +
		"<o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->" +
		"</head>" +
		"<body style=\"margin:0;padding:0;width:100%;background-color:" + emailPage + ";" +
		"-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;\">" +
		// Hidden preview text shown in the inbox list.
		"<div style=\"display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;" +
		"mso-hide:all;font-size:1px;line-height:1px;\">" + emailEscape(preheader) + "</div>" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;background-color:" + emailPage + ";border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:32px 12px 44px 12px;\">" +
		"<!--[if mso]><table role=\"presentation\" width=\"600\" cellpadding=\"0\" cellspacing=\"0\" " +
		"border=\"0\"><tr><td><![endif]-->" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;max-width:600px;margin:0 auto;border-collapse:collapse;\">" +
		emailHeaderBlock(siteURL) +
		emailContentBlock(body) +
		"</table>" +
		emailFooterBlock(siteURL) +
		"<!--[if mso]></td></tr></table><![endif]-->" +
		"</td></tr></table>" +
		"</body></html>"
}

// White header: the logo on the left, with the brand name in the brand red
// beside it. The logo is a fixed square, so it is never stretched.
func emailHeaderBlock(siteURL string) string {
	return "<tr><td style=\"background:" + emailCard + ";border:1px solid " + emailBorder + ";border-bottom:1px solid " +
		emailBorder + ";border-radius:12px 12px 0 0;padding:22px 32px;font-family:" + emailFont + ";\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"border-collapse:collapse;\"><tr>" +
		"<td valign=\"middle\" style=\"vertical-align:middle;padding:0 12px 0 0;width:44px;\">" +
		// Decorative: the brand name beside it is the readable text, so a
		// blocked image leaves no clipped alt text behind.
		"<img src=\"" + emailImageSource(emailLogoCID) + "\" width=\"44\" height=\"44\" alt=\"\" " +
		"style=\"display:block;width:44px;height:44px;border:0;outline:none;text-decoration:none;\">" +
		"</td>" +
		"<td valign=\"middle\" style=\"vertical-align:middle;font-family:" + emailFont + ";font-size:22px;" +
		"line-height:44px;font-weight:700;letter-spacing:2px;color:" + emailBrandRed + ";\">ELVOVE</td>" +
		"</tr></table>" +
		"</td></tr>"
}

// The static QRIS code, served from the same public folder as the logo.
func emailQRISURL(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + "/qris.png"
}

// White body of the card.
func emailContentBlock(body string) string {
	return "<tr><td style=\"background:" + emailCard + ";border:1px solid " + emailBorder + ";border-top:none;" +
		"border-radius:0 0 12px 12px;padding:34px 34px 38px 34px;font-family:" + emailFont + ";\">" +
		body +
		"</td></tr>"
}

// Centred status badge.
func emailBadge(text string) string {
	return "<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" align=\"center\" " +
		"style=\"margin:0 auto;\"><tr>" +
		"<td style=\"background:" + emailBadgeBg + ";border-radius:999px;padding:8px 14px;" +
		"font-family:" + emailFont + ";font-size:10px;line-height:14px;font-weight:700;letter-spacing:1.5px;" +
		"color:" + emailOnInk + ";text-align:center;white-space:nowrap;\">" +
		emailEscape(text) +
		"</td></tr></table>"
}

// Centred headline with its supporting line.
func emailHeadingBlock(headline, description string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 0 0 0;font-family:" + emailFont + ";font-size:25px;" +
		"line-height:35px;font-weight:400;color:" + emailInk + ";text-align:center;\">" +
		emailEscape(headline) +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:12px 0 0 0;font-family:" + emailFont + ";font-size:14px;" +
		"line-height:23px;color:" + emailInk + ";text-align:center;\">" +
		emailEscape(description) +
		"</td></tr>" +
		"</table>"
}

// Left-aligned, personalised greeting.
func emailGreetingBlock(greeting, paragraph string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:30px 0 0 0;font-family:" + emailFont + ";font-size:14px;line-height:24px;" +
		"color:" + emailInk + ";\">" +
		emailEscape(greeting) +
		"</td></tr>" +
		"<tr><td style=\"padding:8px 0 0 0;font-family:" + emailFont + ";font-size:14px;line-height:24px;" +
		"color:" + emailInk + ";\">" +
		emailEscape(paragraph) +
		"</td></tr>" +
		"</table>"
}

// The booking card: a bordered white panel holding a two-column table.
func emailDetailsCard(b bookingEmail) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:30px 0 0 0;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;\">" +
		"<tr><td colspan=\"2\" style=\"padding:18px 18px 4px 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailInk + ";\">" +
		"DETAIL BOOKING" +
		"</td></tr>" +
		emailDetailRow("Nama Pemesan", emailOneLine(b.Name), false, false) +
		emailDetailRow("Tanggal Booking", emailTanggal(b.EventDate), false, false) +
		emailDetailRow("Waktu Sesi", emailJam(b.EventTime), false, false) +
		emailDetailRow("Lokasi", emailOneLine(b.EventLocation), false, false) +
		emailDetailRow("Paket", emailOneLine(b.PackageTitle), false, false) +
		emailDetailRow("Total Harga", emailRupiah(b.PackagePrice), false, true) +
		// The down payment is half the package price, so it is derived rather
		// than stored; without a price both figures stay pending.
		emailDetailRow("DP 50% (Uang Muka)", emailDownPayment(b.PackagePrice), false, true) +
		emailDetailRow("Sisa Pembayaran", emailRemainingBalance(b.PackagePrice), true, true) +
		"</table>" +
		"</td></tr></table>"
}

// The payment card: what to pay now, the account to pay it into, and the
// static QRIS code. Everything is centred, which keeps the short values —
// an account number, a scan code — easy to read on a phone.
func emailPaymentBlock(b bookingEmail, siteURL string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:18px 0 0 0;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;background:" + emailCard + ";\">" +
		"<tr><td align=\"center\" style=\"padding:20px 18px 0 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailInk + ";\">" +
		"PEMBAYARAN" +
		"</td></tr>" +
		// The amount to pay now, the one figure the customer must read. The
		// inset border and the larger type are what set it apart, now that
		// colour is gone.
		"<tr><td align=\"center\" style=\"padding:14px 18px 0 18px;\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" align=\"center\" " +
		"style=\"margin:0 auto;\">" +
		"<tr><td align=\"center\" style=\"background:" + emailInfoBg + ";border:1px solid " + emailBorder +
		";border-radius:8px;padding:14px 26px;font-family:" + emailFont + ";\">" +
		"<div style=\"font-size:11px;line-height:16px;letter-spacing:1.2px;font-weight:700;" +
		"color:" + emailInk + ";\">DP 50% DIBAYAR SEKARANG</div>" +
		"<div style=\"padding:4px 0 0 0;font-size:24px;line-height:32px;font-weight:700;" +
		"color:" + emailInk + ";\">" + emailDownPayment(b.PackagePrice) + "</div>" +
		"</td></tr></table>" +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:16px 18px 0 18px;font-family:" + emailFont + ";font-size:12px;" +
		"line-height:18px;color:" + emailInk + ";\">Transfer BCA</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:2px 18px 0 18px;font-family:" + emailFont + ";font-size:22px;" +
		"line-height:30px;font-weight:700;letter-spacing:1px;color:" + emailInk + ";\">" +
		emailBankAccount +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:2px 18px 0 18px;font-family:" + emailFont + ";font-size:12px;" +
		"line-height:18px;color:" + emailInk + ";\">a.n. " + emailBankHolder + "</td></tr>" +
		// The QRIS code is a static image, so it is loaded from the site the
		// same way the header logo is; data: URIs are stripped by Gmail.
		"<tr><td align=\"center\" style=\"padding:16px 18px 0 18px;font-family:" + emailFont + ";font-size:12px;" +
		"line-height:18px;color:" + emailInk + ";\">atau scan QRIS</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:10px 18px 0 18px;\">" +
		"<img src=\"" + emailImageSource(emailQRISCID) + "\" width=\"220\" alt=\"QRIS Elvove\" " +
		"style=\"display:block;width:220px;max-width:100%;height:auto;border:0;outline:none;" +
		"text-decoration:none;margin:0 auto;\">" +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:8px 18px 20px 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:17px;color:" + emailInk + ";\">" +
		emailQRISMerchant + " &middot; NMID " + emailQRISNMID +
		"</td></tr>" +
		"</table>" +
		"</td></tr></table>"
}

const (
	emailBankAccount  = "8025325911"
	emailBankHolder   = "Leonardus Seco Patu Sila Anggara"
	emailQRISMerchant = "ELVOVE - ART"
	emailQRISNMID     = "ID1026639151079"
)

// The admin booking card: customer contact, then the event details.
func emailAdminDetailsCard(b bookingEmail) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;margin-top:30px;\">" +
		"<tr><td colspan=\"2\" style=\"padding:18px 18px 4px 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailInk + ";\">" +
		"DETAIL BOOKING" +
		"</td></tr>" +
		emailDetailRow("Nama Pemesan", emailOneLine(b.Name), false, false) +
		emailDetailRow("Email", emailOneLine(b.Email), false, false) +
		emailDetailRow("Telepon", emailOneLine(b.Phone), false, false) +
		emailDetailRow("Tanggal Booking", emailTanggal(b.EventDate), false, false) +
		emailDetailRow("Waktu Sesi", emailJam(b.EventTime), false, false) +
		emailDetailRow("Lokasi", emailOneLine(b.EventLocation), false, false) +
		emailDetailRow("Paket", emailOneLine(b.PackageTitle), false, false) +
		emailDetailRow("Total Harga", emailRupiah(b.PackagePrice), false, true) +
		emailDetailRow("Status", emailStatusLabel(b.Status), true, false) +
		"</table>"
}

// emailAdminButton is the button to the booking in the admin panel.
func emailAdminButton(link string) string {
	href := emailEscape(link)
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">" +
		"<tr><td align=\"center\" bgcolor=\"" + emailBadgeBg + "\" style=\"background:" + emailBadgeBg +
		";border-radius:5px;\">" +
		"<!--[if mso]>" +
		"<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" " +
		"xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" " +
		"style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" " +
		"fillcolor=\"" + emailBadgeBg + "\">" +
		"<w:anchorlock/>" +
		"<center style=\"color:#ffffff;font-family:Arial,sans-serif;font-size:12px;font-weight:bold;" +
		"letter-spacing:1px;\">BUKA DI PANEL ADMIN</center>" +
		"</v:roundrect>" +
		"<![endif]-->" +
		"<!--[if !mso]><!-- -->" +
		"<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;" +
		"font-family:" + emailFont + ";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;" +
		"color:#FFFFFF;text-decoration:none;\">BUKA DI PANEL ADMIN</a>" +
		"<!--<![endif]-->" +
		"</td></tr></table>" +
		"</td></tr></table>"
}

// One row of the two-column table: the label on the left, the value on the
// right.
func emailDetailRow(label, value string, last, price bool) string {
	border := ""
	if !last {
		border = "border-bottom:1px solid " + emailBorder + ";"
	}
	valueColor := emailInk
	valueWeight := "400"
	if price {
		valueWeight = "600"
	}
	// The closing row carries the card's bottom breathing room.
	pad := "padding:12px 18px;"
	if last {
		pad = "padding:12px 18px 20px 18px;"
	}
	return "<tr>" +
		"<td width=\"40%\" valign=\"top\" style=\"width:40%;" + pad + border +
		"font-family:" + emailFont + ";font-size:13px;line-height:19px;color:" + emailInk + ";\">" +
		emailEscape(label) +
		"</td>" +
		"<td valign=\"top\" style=\"" + pad + border + "font-family:" + emailFont + ";font-size:13px;" +
		"line-height:19px;font-weight:" + valueWeight + ";color:" + valueColor + ";text-align:right;\">" +
		emailEscape(value) +
		"</td>" +
		"</tr>"
}

// The next-steps box: the same white bordered panel as the booking details,
// with no accent bar, so the message stays monochrome.
func emailNextStepsBlock() string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;background:" + emailCard + ";\">" +
		"<tr>" +
		"<td style=\"padding:18px;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td colspan=\"2\" style=\"padding:0 0 10px 0;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailInk + ";\">" +
		"LANGKAH SELANJUTNYA" +
		"</td></tr>" +
		emailStepRow("Bayar DP 50% lewat transfer BCA atau QRIS.") +
		emailStepRow("Datang tepat waktu di lokasi yang disepakati.") +
		emailStepRow("Perlu ubah jadwal atau ada pertanyaan? Hubungi kami via WhatsApp.") +
		"</table>" +
		"</td>" +
		"</tr>" +
		"</table>" +
		"</td></tr></table>"
}

// One bullet of the next-steps box.
func emailStepRow(step string) string {
	return "<tr>" +
		"<td width=\"14\" valign=\"top\" style=\"width:14px;padding:0 0 8px 0;font-family:" + emailFont +
		";font-size:13px;line-height:22px;color:" + emailInk + ";\">&bull;</td>" +
		"<td valign=\"top\" style=\"padding:0 0 8px 0;font-family:" + emailFont + ";font-size:13px;" +
		"line-height:22px;color:" + emailInk + ";\">" +
		emailEscape(step) +
		"</td>" +
		"</tr>"
}

// WhatsApp button. The number is the one in the site's contact settings, so the
// button opens the same chat as the floating button on the site. When it is
// unset the button points at the site rather than at a dead wa.me link.
func emailWhatsAppButton(number, siteURL string) string {
	href := emailWhatsAppURL(number)
	if href == "" {
		href = emailEscape(siteURL)
	}
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">" +
		"<tr><td align=\"center\" bgcolor=\"" + emailWhatsAppGreen + "\" style=\"background:" +
		emailWhatsAppGreen + ";border-radius:5px;\">" +
		"<!--[if mso]>" +
		"<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" " +
		"xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" " +
		"style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" " +
		"fillcolor=\"" + emailWhatsAppGreen + "\">" +
		"<w:anchorlock/>" +
		"<center style=\"color:#ffffff;font-family:Arial,sans-serif;font-size:12px;font-weight:bold;" +
		"letter-spacing:1px;\">CHAT VIA WHATSAPP</center>" +
		"</v:roundrect>" +
		"<![endif]-->" +
		"<!--[if !mso]><!-- -->" +
		"<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;" +
		"font-family:" + emailFont + ";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;" +
		"color:#FFFFFF;text-decoration:none;\">CHAT VIA WHATSAPP</a>" +
		"<!--<![endif]-->" +
		"</td></tr></table>" +
		"</td></tr></table>"
}

// emailWhatsAppURL turns the stored number into a wa.me link, applying the same
// rule as the site: keep the digits and turn a leading 0 into 62.
func emailWhatsAppURL(number string) string {
	digits := emailNonDigits.ReplaceAllString(emailOneLine(number), "")
	if digits == "" {
		return ""
	}
	if strings.HasPrefix(digits, "0") {
		digits = "62" + digits[1:]
	}
	return "https://wa.me/" + digits
}

// Minimalist centred footer.
func emailFooterBlock(siteURL string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;max-width:600px;margin:0 auto;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 16px 0 16px;font-family:" + emailFont + ";\">" +
		"<div style=\"font-size:11px;line-height:16px;font-weight:700;letter-spacing:2px;color:" + emailInk +
		";\">Elvove</div>" +
		"<div style=\"font-size:11px;line-height:18px;color:" + emailInk + ";padding-top:6px;\">" +
		emailTagline +
		"</div>" +
		"<div style=\"font-size:10px;line-height:16px;color:" + emailInk + ";padding-top:6px;\">" +
		"© " + strconv.Itoa(time.Now().Year()) + " " + emailBrandFallback + ". All rights reserved." +
		"</div>" +
		"<div style=\"font-size:10px;line-height:16px;color:" + emailInk + ";padding-top:4px;\">" +
		"<a href=\"" + emailEscape(siteURL) + "\" style=\"color:" + emailInk + ";text-decoration:underline;\">" +
		emailEscape(siteURL) +
		"</a>" +
		"</div>" +
		"</td></tr></table>"
}

// -------------------------------------------------------------- details

func emailDetailLines(b bookingEmail) string {
	return "Nama Pemesan    : " + emailOneLine(b.Name) + "\n" +
		"Tanggal Booking : " + emailTanggal(b.EventDate) + "\n" +
		"Waktu Sesi      : " + emailJam(b.EventTime) + "\n" +
		"Lokasi          : " + emailOneLine(b.EventLocation) + "\n" +
		"Paket           : " + emailOneLine(b.PackageTitle) + "\n" +
		"Total Harga     : " + emailRupiah(b.PackagePrice) + "\n" +
		"DP 50%          : " + emailDownPayment(b.PackagePrice) + "\n" +
		"Sisa Pembayaran : " + emailRemainingBalance(b.PackagePrice) + "\n"
}

func emailAdminDetailLines(b bookingEmail) string {
	return "Nama Pemesan    : " + emailOneLine(b.Name) + "\n" +
		"Email           : " + emailOneLine(b.Email) + "\n" +
		"Telepon         : " + emailOneLine(b.Phone) + "\n" +
		"Tanggal Booking : " + emailTanggal(b.EventDate) + "\n" +
		"Waktu Sesi      : " + emailJam(b.EventTime) + "\n" +
		"Lokasi          : " + emailOneLine(b.EventLocation) + "\n" +
		"Paket           : " + emailOneLine(b.PackageTitle) + "\n" +
		"Total Harga     : " + emailRupiah(b.PackagePrice) + "\n" +
		"Status          : " + emailStatusLabel(b.Status) + "\n"
}

// emailStatusLabel mirrors BookingEmailTemplate.statusLabel.
func emailStatusLabel(status string) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return emailPending
}

func emailStepsText() string {
	return "• Bayar DP 50% lewat transfer BCA atau QRIS.\n" +
		"• Datang tepat waktu di lokasi yang disepakati.\n" +
		"• Perlu ubah jadwal atau ada pertanyaan? Hubungi kami via WhatsApp.\n"
}

// emailPaymentLines is the plain-text twin of emailPaymentBlock.
func emailPaymentLines(price, siteURL string) string {
	return "PEMBAYARAN\n" +
		"DP 50%          : " + emailDownPayment(price) + " (dibayar sekarang)\n" +
		"Transfer BCA    : " + emailBankAccount + "\n" +
		"Atas Nama       : " + emailBankHolder + "\n" +
		"QRIS            : " + emailQRISURL(siteURL) + "\n" +
		"                  " + emailQRISMerchant + ", NMID " + emailQRISNMID + "\n"
}

func emailPreview(b bookingEmail, headline string) string {
	return headline + " — " + emailOneLine(b.Name) + ". " +
		"Sesi foto Anda pada " + emailTanggal(b.EventDate) +
		". Terima kasih telah memilih Elvove."
}

// -------------------------------------------------------------- helpers

func emailBrandName(senderName string) string {
	name := emailOneLine(senderName)
	if name == "" {
		return emailBrandFallback
	}
	return name
}

// emailTanggal renders a booking date as an Indonesian long date, for example
// "9 November 2026", or the pending marker when it is missing.
func emailTanggal(value string) string {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return emailPending
	}
	return strconv.Itoa(date.Day()) + " " + emailMonths[date.Month()-1] + " " + strconv.Itoa(date.Year())
}

// emailJam renders a booking time as "HH:mm", or the pending marker when it is
// missing.
func emailJam(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return emailPending
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if clock, err := time.Parse(layout, value); err == nil {
			return clock.Format("15:04")
		}
	}
	return emailPending
}

// emailRupiah renders an amount as Indonesian Rupiah with thousand separators,
// for example "Rp200.000", matching Java's "#,##0" format in the id-ID locale.
func emailRupiah(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return emailPending
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return emailPending
	}
	// Java's DecimalFormat default is HALF_EVEN, the same as RoundToEven.
	return "Rp" + emailGroupDigits(int64(math.RoundToEven(amount)))
}

// emailDownPayment renders half the package price, the DP the customer pays to
// secure the session. Java left this row pending; the amount is now fixed at
// 50%, so it is derived from the price.
func emailDownPayment(price string) string {
	amount, ok := emailAmount(price)
	if !ok {
		return emailPending
	}
	return "Rp" + emailGroupDigits(int64(math.RoundToEven(amount/2)))
}

// emailRemainingBalance is the other half, settled after the session.
func emailRemainingBalance(price string) string {
	amount, ok := emailAmount(price)
	if !ok {
		return emailPending
	}
	total := int64(math.RoundToEven(amount))
	down := int64(math.RoundToEven(amount / 2))
	return "Rp" + emailGroupDigits(total-down)
}

// emailAmount parses a package price, reporting whether it holds a number.
func emailAmount(price string) (float64, bool) {
	amount, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
	if err != nil {
		return 0, false
	}
	return amount, true
}

func emailGroupDigits(number int64) string {
	digits := strconv.FormatInt(number, 10)
	negative := strings.HasPrefix(digits, "-")
	if negative {
		digits = digits[1:]
	}
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}

// emailEscape escapes the characters that would otherwise be able to break out
// of a text or an attribute context.
func emailEscape(value string) string {
	if value == "" {
		return ""
	}
	var out strings.Builder
	out.Grow(len(value) + 16)
	for _, r := range value {
		switch r {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		case '"':
			out.WriteString("&quot;")
		case '\'':
			out.WriteString("&#39;")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// emailOneLine collapses line breaks so a value cannot break the layout.
func emailOneLine(value string) string {
	return strings.TrimSpace(emailLineBreaks.ReplaceAllString(value, " "))
}

// PreviewConfirmation renders a sample confirmation e-mail; used by
// cmd/previewemail to eyeball the layout outside a mail client.
func PreviewConfirmation() (string, string, string) {
	return renderBookingConfirmation(bookingEmail{
		Name:          "Budi Santoso",
		EventDate:     "2026-11-09",
		EventTime:     "09:30",
		EventLocation: "Jakarta Selatan",
		PackageTitle:  "Premium",
		PackagePrice:  "2000000.00",
	}, "Elvove", "081234567890", "https://elvove.site")
}

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

	// Palette: white header and card on an off-white page, the site's brand red
	// for the brand name and accents.
	emailPage     = "#F8F7FB"
	emailCard     = "#FFFFFF"
	emailDark     = "#171525"
	emailBrandRed = "#E11D48"
	emailText     = "#292638"
	emailMuted    = "#858191"
	emailInfoBg   = "#F4F1F8"
	emailBorder   = "#EEEAF2"
	emailBadgeBg  = "#FCEEF1"

	emailBrandFallback = "Elvove"
	emailTagline       = "Capturing Moments, Creating Memories."
	emailPending       = "Akan dikonfirmasi"
	emailSiteFallback  = "https://elvove.site"
)

var emailLineBreaks = regexp.MustCompile(`[\r\n\t]+`)

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
func renderBookingConfirmation(b bookingEmail, senderName, senderEmail, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	contact := emailContactEmail(senderEmail)
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
		emailNextStepsBlock() +
		emailContactButton(contact, siteURL)

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
	text.WriteString("\nLANGKAH SELANJUTNYA\n")
	text.WriteString(emailStepsText())
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
func renderBookingReschedule(b bookingEmail, senderName, senderEmail, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	contact := emailContactEmail(senderEmail)
	customer := emailOneLine(b.Name)

	body := emailBadge("PERUBAHAN JADWAL") +
		emailHeadingBlock(
			"Jadwal Sesi Foto Diperbarui",
			"Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". Kami ingin segera menginformasikan usulan perubahan "+
				"jadwal sesi foto Anda. Mohon balas email ini untuk mengonfirmasi atau berdiskusi.") +
		emailDetailsCard(b) +
		emailNextStepsBlock() +
		emailContactButton(contact, siteURL)

	var text strings.Builder
	text.WriteString("PERUBAHAN JADWAL\n\n")
	text.WriteString("Jadwal Sesi Foto Diperbarui\n\n")
	text.WriteString("Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". ")
	text.WriteString("Kami ingin segera menginformasikan usulan perubahan jadwal sesi foto Anda. ")
	text.WriteString("Mohon balas email ini untuk mengonfirmasi atau berdiskusi.\n\n")
	text.WriteString("DETAIL BOOKING\n")
	text.WriteString(emailDetailLines(b))
	text.WriteString("\nLANGKAH SELANJUTNYA\n")
	text.WriteString(emailStepsText())
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
		"<img src=\"" + emailEscape(emailLogoURL(siteURL)) + "\" width=\"44\" height=\"44\" alt=\"\" " +
		"style=\"display:block;width:44px;height:44px;border:0;outline:none;text-decoration:none;\">" +
		"</td>" +
		"<td valign=\"middle\" style=\"vertical-align:middle;font-family:" + emailFont + ";font-size:22px;" +
		"line-height:44px;font-weight:700;letter-spacing:2px;color:" + emailBrandRed + ";\">ELVOVE</td>" +
		"</tr></table>" +
		"</td></tr>"
}

// The public logo on the site; email clients load it over HTTPS.
func emailLogoURL(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + "/logo.png"
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
		"color:" + emailBrandRed + ";text-align:center;white-space:nowrap;\">" +
		emailEscape(text) +
		"</td></tr></table>"
}

// Centred headline with its supporting line.
func emailHeadingBlock(headline, description string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 0 0 0;font-family:" + emailFont + ";font-size:25px;" +
		"line-height:35px;font-weight:400;color:" + emailText + ";text-align:center;\">" +
		emailEscape(headline) +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:12px 0 0 0;font-family:" + emailFont + ";font-size:14px;" +
		"line-height:23px;color:" + emailMuted + ";text-align:center;\">" +
		emailEscape(description) +
		"</td></tr>" +
		"</table>"
}

// Left-aligned, personalised greeting.
func emailGreetingBlock(greeting, paragraph string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:30px 0 0 0;font-family:" + emailFont + ";font-size:14px;line-height:24px;" +
		"color:" + emailText + ";\">" +
		emailEscape(greeting) +
		"</td></tr>" +
		"<tr><td style=\"padding:8px 0 0 0;font-family:" + emailFont + ";font-size:14px;line-height:24px;" +
		"color:" + emailMuted + ";\">" +
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
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailMuted + ";\">" +
		"DETAIL BOOKING" +
		"</td></tr>" +
		emailDetailRow("Nama Pemesan", emailOneLine(b.Name), false, false) +
		emailDetailRow("Tanggal Booking", emailTanggal(b.EventDate), false, false) +
		emailDetailRow("Waktu Sesi", emailJam(b.EventTime), false, false) +
		emailDetailRow("Lokasi", emailOneLine(b.EventLocation), false, false) +
		emailDetailRow("Paket", emailOneLine(b.PackageTitle), false, false) +
		emailDetailRow("Total Harga", emailRupiah(b.PackagePrice), false, true) +
		// The down payment and the remaining balance are agreed with the
		// client, so they are shown as pending rather than as a figure the
		// booking does not hold.
		emailDetailRow("DP (Uang Muka)", emailPending, false, true) +
		emailDetailRow("Sisa Pembayaran", emailPending, true, true) +
		"</table>" +
		"</td></tr></table>"
}

// The admin booking card: customer contact, then the event details.
func emailAdminDetailsCard(b bookingEmail) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;margin-top:30px;\">" +
		"<tr><td colspan=\"2\" style=\"padding:18px 18px 4px 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailMuted + ";\">" +
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
		"<tr><td align=\"center\" bgcolor=\"" + emailBrandRed + "\" style=\"background:" + emailBrandRed +
		";border-radius:5px;\">" +
		"<!--[if mso]>" +
		"<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" " +
		"xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" " +
		"style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" " +
		"fillcolor=\"" + emailBrandRed + "\">" +
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
	valueColor := emailText
	valueWeight := "400"
	if price {
		valueColor = emailBrandRed
		valueWeight = "600"
	}
	// The closing row carries the card's bottom breathing room.
	pad := "padding:12px 18px;"
	if last {
		pad = "padding:12px 18px 20px 18px;"
	}
	return "<tr>" +
		"<td width=\"40%\" valign=\"top\" style=\"width:40%;" + pad + border +
		"font-family:" + emailFont + ";font-size:13px;line-height:19px;color:" + emailMuted + ";\">" +
		emailEscape(label) +
		"</td>" +
		"<td valign=\"top\" style=\"" + pad + border + "font-family:" + emailFont + ";font-size:13px;" +
		"line-height:19px;font-weight:" + valueWeight + ";color:" + valueColor + ";text-align:right;\">" +
		emailEscape(value) +
		"</td>" +
		"</tr>"
}

// The highlighted next-steps box. The accent bar is a table cell rather than a
// border, because Outlook drops borders on cells.
func emailNextStepsBlock() string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr>" +
		"<td width=\"3\" style=\"width:3px;background:" + emailBrandRed + ";border-radius:6px 0 0 6px;" +
		"font-size:0;line-height:0;\">&nbsp;</td>" +
		"<td style=\"background:" + emailInfoBg + ";border-radius:0 6px 6px 0;padding:18px;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td colspan=\"2\" style=\"padding:0 0 10px 0;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailBrandRed + ";\">" +
		"LANGKAH SELANJUTNYA" +
		"</td></tr>" +
		emailStepRow("Tim kami akan menghubungi Anda untuk mengonfirmasi down payment (DP) dan menyepakati "+
			"jadwal sesi foto.") +
		emailStepRow("Mohon siapkan konfirmasi mengenai paket, durasi, dan kebutuhan khusus untuk sesi Anda.") +
		emailStepRow("Datang tepat waktu di lokasi yang telah disepakati agar sesi berjalan optimal.") +
		emailStepRow("Punya pertanyaan atau perlu mengubah jadwal? Balas email ini, tim kami siap membantu.") +
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
		";font-size:13px;line-height:22px;color:" + emailBrandRed + ";\">&bull;</td>" +
		"<td valign=\"top\" style=\"padding:0 0 8px 0;font-family:" + emailFont + ";font-size:13px;" +
		"line-height:22px;color:" + emailText + ";\">" +
		emailEscape(step) +
		"</td>" +
		"</tr>"
}

// Reply button. The address is the one the application already uses.
func emailContactButton(contact, siteURL string) string {
	// The sender address is configured in the application and is never
	// hardcoded here. If it is ever missing, the button points at the site
	// rather than at a broken "mailto:" with no address.
	href := emailEscape(siteURL)
	if contact != "" {
		href = "mailto:" + emailEscape(contact)
	}
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">" +
		"<tr><td align=\"center\" bgcolor=\"" + emailDark + "\" style=\"background:" + emailDark +
		";border-radius:5px;\">" +
		"<!--[if mso]>" +
		"<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" " +
		"xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" " +
		"style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" " +
		"fillcolor=\"" + emailDark + "\">" +
		"<w:anchorlock/>" +
		"<center style=\"color:#ffffff;font-family:Arial,sans-serif;font-size:12px;font-weight:bold;" +
		"letter-spacing:1px;\">BALAS EMAIL KAMI</center>" +
		"</v:roundrect>" +
		"<![endif]-->" +
		"<!--[if !mso]><!-- -->" +
		"<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;" +
		"font-family:" + emailFont + ";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;" +
		"color:#FFFFFF;text-decoration:none;\">BALAS EMAIL KAMI</a>" +
		"<!--<![endif]-->" +
		"</td></tr></table>" +
		"</td></tr></table>"
}

// Minimalist centred footer.
func emailFooterBlock(siteURL string) string {
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;max-width:600px;margin:0 auto;border-collapse:collapse;\">" +
		"<tr><td align=\"center\" style=\"padding:26px 16px 0 16px;font-family:" + emailFont + ";\">" +
		"<div style=\"font-size:11px;line-height:16px;font-weight:700;letter-spacing:2px;color:" + emailMuted +
		";\">Elvove</div>" +
		"<div style=\"font-size:11px;line-height:18px;color:" + emailMuted + ";padding-top:6px;\">" +
		emailTagline +
		"</div>" +
		"<div style=\"font-size:10px;line-height:16px;color:" + emailMuted + ";padding-top:6px;\">" +
		"© " + strconv.Itoa(time.Now().Year()) + " " + emailBrandFallback + ". All rights reserved." +
		"</div>" +
		"<div style=\"font-size:10px;line-height:16px;color:" + emailMuted + ";padding-top:4px;\">" +
		"<a href=\"" + emailEscape(siteURL) + "\" style=\"color:" + emailMuted + ";text-decoration:underline;\">" +
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
		"DP (Uang Muka)  : " + emailPending + "\n" +
		"Sisa Pembayaran : " + emailPending + "\n"
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
	return "• Tim kami akan menghubungi Anda untuk mengonfirmasi down payment (DP) dan menyepakati jadwal " +
		"sesi foto.\n" +
		"• Mohon siapkan konfirmasi mengenai paket, durasi, dan kebutuhan khusus untuk sesi Anda.\n" +
		"• Datang tepat waktu di lokasi yang telah disepakati agar sesi berjalan optimal.\n" +
		"• Punya pertanyaan atau perlu mengubah jadwal? Balas email ini, tim kami siap membantu.\n"
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

func emailContactEmail(senderEmail string) string {
	return emailOneLine(senderEmail)
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
	}, "Elvove", "elvo.vee@gmail.com", "https://elvove.site")
}

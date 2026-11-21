package httpapi

// Booking e-mail templates. This is a direct port of the Java
// BookingEmailTemplate: table-based markup with inline CSS only, so Gmail,
// Outlook, Apple Mail and mobile clients render the same layout. Every value
// that comes from a booking is escaped before it is interpolated, so customer
// input can never inject markup into the message.

import (
	"fmt"
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

	// Palette: black ink on white. The logo, the ELVOVE wordmark and the call
	// to action buttons carry a colour; the WhatsApp button keeps its own
	// green, everything else stays monochrome.
	emailPage     = "#F5F5F5"
	emailCard     = "#FFFFFF"
	emailInk      = "#000000"
	emailOnInk    = "#FFFFFF"
	emailBrandRed = "#E11D48" // the wordmark and every button except WhatsApp
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
		emailPaymentBlock(b) +
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

// invoiceDetails is what the invoice adds to the booking itself: the number
// that identifies the transaction, when it was issued, the payment points the
// admin recorded, and the Drive folder holding the edited photos.
type invoiceDetails struct {
	Number      string // INV-2026-0012
	IssuedAt    string // YYYY-MM-DD
	DownPaidAt  string // YYYY-MM-DD; empty while no payment is recorded
	SettledAt   string // YYYY-MM-DD; empty while the balance is open
	DeliveryURL string // Google Drive folder with the edited photos
	// GalleryURL is the client's photo-selection page, offered by its own
	// notice before the settlement one, so the settlement e-mail only asks for
	// money.
	GalleryURL     string
	SelectionLimit int
}

// paymentStateLabel is the one line the messages show about what has arrived.
func (d invoiceDetails) paymentStateLabel() string {
	switch {
	case d.settled():
		return "Lunas pada " + emailTanggal(d.SettledAt)
	case d.downPaid():
		return "DP dibayar pada " + emailTanggal(d.DownPaidAt)
	default:
		return "Belum ada pembayaran"
	}
}

func (d invoiceDetails) settled() bool  { return strings.TrimSpace(d.SettledAt) != "" }
func (d invoiceDetails) downPaid() bool { return strings.TrimSpace(d.DownPaidAt) != "" }

// renderBookingInvoice is the last message of a booking: the transaction as it
// was paid, and the link to the edited photos. The confirmation asks for the DP
// and the settlement notice for the rest, so this one asks for nothing and only
// closes the order.
func renderBookingInvoice(b bookingEmail, invoice invoiceDetails, senderName, whatsapp, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)
	issued := emailTanggal(invoice.IssuedAt)

	body := emailBadge("INVOICE") +
		emailHeadingBlock(
			"Invoice & Hasil Foto",
			"Nomor "+invoice.Number+" · diterbitkan "+issued+".") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". Berikut rincian transaksi pemesanan Anda, "+
				"beserta tautan foto yang sudah diedit.") +
		// The booking card leaves out the amounts: this message is a receipt,
		// so the money sits in its own transaction panel below.
		emailCardPanel("DETAIL BOOKING", []detailLine{
			{Label: "Nama Pemesan", Value: emailOneLine(b.Name)},
			{Label: "Tanggal Booking", Value: emailTanggal(b.EventDate)},
			{Label: "Waktu Sesi", Value: emailJam(b.EventTime)},
			{Label: "Lokasi", Value: emailOneLine(b.EventLocation)},
			{Label: "Paket", Value: emailOneLine(b.PackageTitle)},
		}) +
		emailCardPanel("RINCIAN TRANSAKSI", []detailLine{
			{Label: "Nomor Invoice", Value: invoice.Number},
			{Label: "Tanggal Invoice", Value: issued},
			{Label: "Total Harga", Value: emailRupiah(b.PackagePrice), Price: true},
			{Label: "DP 50% Dibayar", Value: emailDownPayment(b.PackagePrice), Price: true},
			{Label: "Pelunasan Dibayar", Value: emailRemainingBalance(b.PackagePrice), Price: true},
			{Label: "Status", Value: invoice.paymentStateLabel()},
		}) +
		emailGalleryBlock(galleryBlock{
			Title:  "FOTO SIAP DIUNDUH",
			Note:   "Seluruh foto yang sudah diedit tersedia di Google Drive.",
			Button: "BUKA GALERI FOTO",
			Link:   invoice.DeliveryURL,
		}) +
		emailWhatsAppButton(whatsapp, siteURL)

	var text strings.Builder
	text.WriteString("INVOICE\n\n")
	text.WriteString("Invoice & Hasil Foto\n\n")
	text.WriteString("Nomor " + invoice.Number + " · diterbitkan " + issued + ".\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". Berikut rincian transaksi pemesanan Anda, ")
	text.WriteString("beserta tautan foto yang sudah diedit.\n\n")
	text.WriteString("DETAIL BOOKING\n")
	text.WriteString("Nama Pemesan    : " + emailOneLine(b.Name) + "\n")
	text.WriteString("Tanggal Booking : " + emailTanggal(b.EventDate) + "\n")
	text.WriteString("Waktu Sesi      : " + emailJam(b.EventTime) + "\n")
	text.WriteString("Lokasi          : " + emailOneLine(b.EventLocation) + "\n")
	text.WriteString("Paket           : " + emailOneLine(b.PackageTitle) + "\n")
	text.WriteString("\nRINCIAN TRANSAKSI\n")
	text.WriteString("Nomor Invoice    : " + invoice.Number + "\n")
	text.WriteString("Tanggal Invoice  : " + issued + "\n")
	text.WriteString("Total Harga      : " + emailRupiah(b.PackagePrice) + "\n")
	text.WriteString("DP 50% Dibayar   : " + emailDownPayment(b.PackagePrice) + "\n")
	text.WriteString("Pelunasan Dibayar: " + emailRemainingBalance(b.PackagePrice) + "\n")
	text.WriteString("Status           : " + invoice.paymentStateLabel() + "\n")
	text.WriteString("\nFOTO SIAP DIUNDUH\n")
	text.WriteString("Foto yang sudah diedit: " + invoice.DeliveryURL + "\n")
	if url := emailWhatsAppURL(whatsapp); url != "" {
		text.WriteString("\nWhatsApp: " + url + "\n")
	}
	text.WriteString("\nElvove\n")
	text.WriteString(emailTagline + "\n")
	text.WriteString("© " + strconv.Itoa(time.Now().Year()) + " " + brand + ". All rights reserved.\n")
	text.WriteString(siteURL + "\n")

	subject := "Invoice " + invoice.Number + " — " + brand
	preview := emailPreview(b, "Invoice "+invoice.Number)
	return subject, emailDocumentTitled(body, preview, siteURL, "Invoice "+invoice.Number+" — Elvove"), text.String()
}

// galleryBlock is one bordered panel: a title, one line of context and a button.
type galleryBlock struct {
	Title  string
	Note   string
	Button string
	Link   string
}

// emailGalleryBlock renders the panel, so the photos are one tap away on a phone.
func emailGalleryBlock(block galleryBlock) string {
	href := emailEscape(block.Link)
	label := block.Button
	return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border-collapse:collapse;\">" +
		"<tr><td style=\"padding:26px 0 0 0;\">" +
		"<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" " +
		"style=\"width:100%;border:1px solid " + emailBorder + ";border-radius:10px;background:" + emailCard + ";\">" +
		"<tr><td align=\"center\" style=\"padding:18px 18px 0 18px;font-family:" + emailFont + ";font-size:11px;" +
		"line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + emailInk + ";\">" +
		emailEscape(block.Title) +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:8px 18px 0 18px;font-family:" + emailFont + ";font-size:13px;" +
		"line-height:20px;color:" + emailInk + ";\">" +
		emailEscape(block.Note) +
		"</td></tr>" +
		"<tr><td align=\"center\" style=\"padding:16px 18px 20px 18px;\">" +
		"<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">" +
		"<tr><td align=\"center\" bgcolor=\"" + emailBrandRed + "\" style=\"background:" + emailBrandRed + ";border-radius:5px;\">" +
		"<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;font-family:" + emailFont +
		";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;color:" + emailOnInk +
		";text-decoration:none;\">" + label + "</a>" +
		"</td></tr></table>" +
		"</td></tr>" +
		"</table>" +
		"</td></tr></table>"
}

// renderBookingSelection is the notice that opens the client page: the photos
// are ready, and this is where the ones to edit are chosen. It is sent before
// the settlement notice, so the choice is made while the photos are fresh, and
// the settlement notice is left to ask for money only.
func renderBookingSelection(b bookingEmail, invoice invoiceDetails, senderName, whatsapp, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)
	title := emailSelectionTitle(invoice.SelectionLimit)

	body := emailBadge("PILIH FOTO") +
		emailHeadingBlock(
			title,
			"Sesi foto Anda pada "+emailTanggal(b.EventDate)+" sudah siap.") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". Foto sesi Anda sudah kami unggah, "+
				"dan yang ingin diedit dipilih lewat tombol di bawah.") +
		emailGalleryBlock(galleryBlock{
			Title:  title,
			Note:   emailSelectionNote(invoice.SelectionLimit),
			Button: emailSelectionButton(invoice.SelectionLimit),
			Link:   invoice.GalleryURL,
		})
	if invoice.SelectionLimit > 0 {
		body += emailListBlock("CARA MEMILIH",
			fmt.Sprintf("Buka halaman foto, lalu tandai maksimal %d foto yang ingin diedit.", invoice.SelectionLimit),
			"Tekan KIRIM PILIHAN FOTO. Setelah dikirim, pilihan tidak dapat diubah.",
			"Hasil edit dikirim ke email ini setelah selesai.")
	}
	body += emailWhatsAppButton(whatsapp, siteURL)

	var text strings.Builder
	text.WriteString("PILIH FOTO\n\n")
	text.WriteString(title + "\n\n")
	text.WriteString("Sesi foto Anda pada " + emailTanggal(b.EventDate) + " sudah siap.\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". Foto sesi Anda sudah kami unggah, ")
	text.WriteString("dan yang ingin diedit dipilih lewat tautan di bawah.\n\n")
	text.WriteString(strings.ToUpper(title) + "\n")
	text.WriteString(emailSelectionNote(invoice.SelectionLimit) + "\n")
	text.WriteString(invoice.GalleryURL + "\n")
	if invoice.SelectionLimit > 0 {
		text.WriteString("\nCARA MEMILIH\n")
		text.WriteString(fmt.Sprintf("• Buka halaman foto, lalu tandai maksimal %d foto yang ingin diedit.\n", invoice.SelectionLimit))
		text.WriteString("• Tekan KIRIM PILIHAN FOTO. Setelah dikirim, pilihan tidak dapat diubah.\n")
		text.WriteString("• Hasil edit dikirim ke email ini setelah selesai.\n")
	}
	if url := emailWhatsAppURL(whatsapp); url != "" {
		text.WriteString("\nWhatsApp: " + url + "\n")
	}
	text.WriteString("\n" + brand + "\n")
	text.WriteString(emailTagline + "\n")
	text.WriteString("© " + strconv.Itoa(time.Now().Year()) + " " + brand + ". All rights reserved.\n")
	text.WriteString(siteURL + "\n")

	subject := emailSelectionSubject(brand)
	preview := emailPreview(b, title)
	return subject, emailDocumentTitled(body, preview, siteURL, subject), text.String()
}

// emailSelectionSubject names the message in the inbox. It says what to do
// rather than what it is about, so it is obvious this one needs the client.
func emailSelectionSubject(brand string) string {
	return "Pilih Foto untuk Diedit — " + brand
}

// renderBookingSettlement is the settlement notice: the remaining balance, the
// account, the QRIS code and the deadline before the session. It deliberately
// skips the package description the confirmation carries, so the one figure the
// customer still owes is what stands out.
func renderBookingSettlement(b bookingEmail, invoice invoiceDetails, senderName, whatsapp, siteURL string) (string, string, string) {
	siteURL = emailSiteURL(siteURL)
	brand := emailBrandName(senderName)
	customer := emailOneLine(b.Name)
	remaining := emailRemainingBalance(b.PackagePrice)
	due := emailSettlementDeadline(b.EventDate, invoice.IssuedAt)

	body := emailBadge("PELUNASAN") +
		emailHeadingBlock(
			"Pelunasan Sisa Pembayaran",
			"Sisa pembayaran sesi foto Anda pada "+emailTanggal(b.EventDate)+".") +
		emailGreetingBlock(
			"Halo, "+customer+".",
			"Terima kasih telah memilih "+brand+". DP untuk sesi foto Anda sudah kami terima. "+
				"Mohon lunasi sisa pembayaran sebesar "+remaining+" "+due+".") +
		emailCardPanel("RINCIAN PELUNASAN", []detailLine{
			{Label: "Nomor Invoice", Value: invoice.Number},
			{Label: "Tanggal Sesi", Value: emailTanggal(b.EventDate)},
			{Label: "Waktu Sesi", Value: emailJam(b.EventTime)},
			{Label: "Total Harga", Value: emailRupiah(b.PackagePrice), Price: true},
			{Label: "DP Sudah Dibayar", Value: emailDownPayment(b.PackagePrice), Price: true},
			{Label: "Sisa Pembayaran", Value: remaining, Price: true},
		}) +
		emailPaymentCard("SISA PEMBAYARAN", remaining) +
		emailListBlock("CARA MELUNASI",
			"Selesaikan pembayaran "+due+".",
			"Kirim bukti transfer via WhatsApp agar pemesanan Anda kami tandai lunas.")
	body += emailWhatsAppButton(whatsapp, siteURL)

	var text strings.Builder
	text.WriteString("PELUNASAN\n\n")
	text.WriteString("Pelunasan Sisa Pembayaran\n\n")
	text.WriteString("Sisa pembayaran sesi foto Anda pada " + emailTanggal(b.EventDate) + ".\n\n")
	text.WriteString("Halo, " + customer + ".\n\n")
	text.WriteString("Terima kasih telah memilih " + brand + ". DP untuk sesi foto Anda sudah kami terima. ")
	text.WriteString("Mohon lunasi sisa pembayaran " + due + ".\n\n")
	text.WriteString("RINCIAN PELUNASAN\n")
	text.WriteString("Nomor Invoice   : " + invoice.Number + "\n")
	text.WriteString("Tanggal Sesi    : " + emailTanggal(b.EventDate) + "\n")
	text.WriteString("Waktu Sesi      : " + emailJam(b.EventTime) + "\n")
	text.WriteString("Total Harga     : " + emailRupiah(b.PackagePrice) + "\n")
	text.WriteString("DP Sudah Dibayar: " + emailDownPayment(b.PackagePrice) + "\n")
	text.WriteString("Sisa Pembayaran : " + remaining + "\n\n")
	text.WriteString(emailPaymentInstructions("Sisa Pembayaran", remaining, siteURL))
	text.WriteString("\nCARA MELUNASI\n")
	text.WriteString("• Selesaikan pembayaran " + due + ".\n")
	text.WriteString("• Kirim bukti transfer via WhatsApp agar pemesanan Anda kami tandai lunas.\n")
	if url := emailWhatsAppURL(whatsapp); url != "" {
		text.WriteString("\nWhatsApp: " + url + "\n")
	}
	text.WriteString("\nElvove\n")
	text.WriteString(emailTagline + "\n")
	text.WriteString("© " + strconv.Itoa(time.Now().Year()) + " " + brand + ". All rights reserved.\n")
	text.WriteString(siteURL + "\n")

	subject := "Pelunasan Sisa Pembayaran — " + brand
	preview := emailPreview(b, "Pelunasan Sisa Pembayaran")
	return subject, emailDocumentTitled(body, preview, siteURL, "Pelunasan Sisa Pembayaran — Elvove"), text.String()
}

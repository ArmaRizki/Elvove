package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.math.BigDecimal;
import java.text.DecimalFormat;
import java.text.DecimalFormatSymbols;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.Year;
import java.time.format.DateTimeFormatter;
import java.util.Locale;

/**
 * Renders the ELVOVE booking e-mails.
 *
 * <p>The document is table-based and styled with inline CSS only: no external
 * stylesheet, no script, no flexbox and no grid. Gmail, Outlook, Apple Mail and
 * the mobile clients therefore render the same layout, and vertical rhythm is
 * controlled with padding on table cells rather than on divs, which Outlook
 * ignores.</p>
 *
 * <p>Every value that comes from the booking is escaped before it is
 * interpolated, so customer input can never inject markup into the message.
 * Each e-mail also carries a plain-text part, which is what a client that
 * cannot display HTML shows.</p>
 */
@Component
public class BookingEmailTemplate {

    /** Subject line, HTML body and plain-text body of one e-mail. */
    public record EmailContent(String subject, String html, String text) {}

    private static final Locale ID = Locale.forLanguageTag("id-ID");
    private static final DateTimeFormatter TANGGAL = DateTimeFormatter.ofPattern("d MMMM yyyy", ID);
    private static final DateTimeFormatter JAM = DateTimeFormatter.ofPattern("HH:mm");

    // Inter from Google Fonts where the client loads web fonts; Arial and
    // Helvetica are used everywhere else, so the text stays readable.
    private static final String FONT = "Inter, Arial, Helvetica, sans-serif";
    private static final String FONT_IMPORT =
            "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap";

    // Palette: white header and card on an off-white page, the site's brand red
    // (--color-brass in frontend/src/index.css) for the brand name and accents.
    private static final String PAGE = "#F8F7FB";
    private static final String CARD = "#FFFFFF";
    private static final String DARK = "#171525";
    private static final String BRAND_RED = "#E11D48";
    private static final String ACCENT = BRAND_RED;
    private static final String TEXT = "#292638";
    private static final String MUTED = "#858191";
    private static final String INFO_BG = "#F4F1F8";
    private static final String BORDER = "#EEEAF2";
    private static final String BADGE_BG = "#FCEEF1";
    private static final String BADGE_TEXT = BRAND_RED;

    private static final String BRAND_FALLBACK = "Elvove";
    private static final String TAGLINE = "Capturing Moments, Creating Memories.";
    private static final String PENDING = "Akan dikonfirmasi";

    private final String siteUrl;

    /**
     * The site address is used for the header logo (a public image URL that
     * email clients can load) and for the links in the footer.
     */
    public BookingEmailTemplate(
            @Value("${app.mail.site-url:https://elvove.site}") String siteUrl) {
        this.siteUrl = siteUrl == null || siteUrl.isBlank() ? "https://elvove.site" : siteUrl.trim();
    }

    /**
     * The booking confirmation: status badge, headline, the personalised
     * greeting, the booking card, the next steps and the reply button.
     */
    public EmailContent confirmation(Booking booking, String senderName, String senderEmail) {
        String brand = brandName(senderName);
        String contact = contactEmail(senderEmail);
        String customer = oneLine(booking.getName());

        String html = badge("PESANAN DITERIMA")
                + headingBlock("Booking Anda Telah Dikonfirmasi",
                "Detail reservasi Anda sudah tercatat. Tim kami akan menghubungi Anda untuk konfirmasi lebih lanjut.")
                + greetingBlock("Halo, " + customer + ".",
                "Terima kasih telah memilih " + brand + ". Kami telah menerima dan mengonfirmasi booking Anda. "
                        + "Berikut adalah detail reservasi Anda.")
                + detailsCard(booking)
                + nextStepsBlock()
                + contactButton(contact);

        String text = new StringBuilder()
                .append("PESANAN DITERIMA").append("\n\n")
                .append("Booking Anda Telah Dikonfirmasi").append("\n\n")
                .append("Detail reservasi Anda sudah tercatat. Tim kami akan menghubungi Anda untuk konfirmasi ")
                .append("lebih lanjut.").append("\n\n")
                .append("Halo, ").append(customer).append(".").append("\n\n")
                .append("Terima kasih telah memilih ").append(brand).append(". ")
                .append("Kami telah menerima dan mengonfirmasi booking Anda. ")
                .append("Berikut adalah detail reservasi Anda.").append("\n\n")
                .append("DETAIL BOOKING").append("\n")
                .append(detailLines(booking)).append("\n")
                .append("LANGKAH SELANJUTNYA").append("\n")
                .append(stepsText()).append("\n")
                .append("Elvove").append("\n")
                .append(TAGLINE).append("\n")
                .append("© ").append(Year.now().getValue()).append(' ').append(brand).append(". All rights reserved.")
                .append("\n").append(siteUrl).append("\n")
                .toString();

        return new EmailContent("Booking Anda Telah Dikonfirmasi — " + brand,
                document(html, preview(booking, "Booking Anda Telah Dikonfirmasi")), text);
    }

    /**
     * The schedule-change notice. It shares the same layout as the confirmation
     * so the customer always sees one consistent ELVOVE design.
     */
    public EmailContent reschedule(Booking booking, String senderName, String senderEmail) {
        String brand = brandName(senderName);
        String contact = contactEmail(senderEmail);
        String customer = oneLine(booking.getName());

        String html = badge("PERUBAHAN JADWAL")
                + headingBlock("Jadwal Sesi Foto Diperbarui",
                "Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.")
                + greetingBlock("Halo, " + customer + ".",
                "Terima kasih telah memilih " + brand + ". Kami ingin segera menginformasikan usulan perubahan "
                        + "jadwal sesi foto Anda. Mohon balas email ini untuk mengonfirmasi atau berdiskusi.")
                + detailsCard(booking)
                + nextStepsBlock()
                + contactButton(contact);

        String text = new StringBuilder()
                .append("PERUBAHAN JADWAL").append("\n\n")
                .append("Jadwal Sesi Foto Diperbarui").append("\n\n")
                .append("Berikut jadwal baru yang kami usulkan untuk sesi foto Anda.").append("\n\n")
                .append("Halo, ").append(customer).append(".").append("\n\n")
                .append("Terima kasih telah memilih ").append(brand).append(". ")
                .append("Kami ingin segera menginformasikan usulan perubahan jadwal sesi foto Anda. ")
                .append("Mohon balas email ini untuk mengonfirmasi atau berdiskusi.").append("\n\n")
                .append("DETAIL BOOKING").append("\n")
                .append(detailLines(booking)).append("\n")
                .append("LANGKAH SELANJUTNYA").append("\n")
                .append(stepsText()).append("\n")
                .append("Elvove").append("\n")
                .append(TAGLINE).append("\n")
                .append("© ").append(Year.now().getValue()).append(' ').append(brand).append(". All rights reserved.")
                .append("\n").append(siteUrl).append("\n")
                .toString();

        return new EmailContent("Jadwal Sesi Foto Diperbarui — " + brand,
                document(html, preview(booking, "Jadwal Sesi Foto Diperbarui")), text);
    }

    /**
     * The administrator notification for a new booking. It lists the booking
     * fields the team needs and links to the booking in the admin panel. It is
     * a separate e-mail from the customer confirmation.
     */
    public EmailContent adminNotification(Booking booking, String senderName, String link) {
        String brand = brandName(senderName);
        String customer = oneLine(booking.getName());
        String id = booking.getId() == null ? "-" : String.valueOf(booking.getId());

        String html = badge("BOOKING BARU")
                + headingBlock("Ada Pemesanan Baru",
                "Sebuah pemesanan baru telah tersimpan di sistem " + brand + ".")
                + greetingBlock("Booking #" + id + " dari " + customer + ".",
                "Berikut detail pemesanan yang baru masuk. Buka panel admin untuk meninjau dan mengubah status.")
                + adminDetailsCard(booking, id)
                + adminButton(link);

        String text = new StringBuilder()
                .append("BOOKING BARU").append("\n\n")
                .append("Ada Pemesanan Baru").append("\n\n")
                .append("Booking #").append(id).append(" dari ").append(customer).append(".").append("\n\n")
                .append("DETAIL BOOKING").append("\n")
                .append(adminDetailLines(booking, id)).append("\n")
                .append("Buka panel admin: ").append(link).append("\n\n")
                .append("Elvove").append("\n")
                .append(TAGLINE).append("\n")
                .toString();

        return new EmailContent("Booking Baru #" + id + " — " + customer,
                document(html, "Booking baru #" + id + " dari " + customer + "."), text);
    }

    // ---------------------------------------------------------------- layout

    private String document(String body, String preheader) {
        return "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" "
                + "\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">"
                + "<html xmlns=\"http://www.w3.org/1999/xhtml\" "
                + "xmlns:v=\"urn:schemas-microsoft-com:vml\" "
                + "xmlns:o=\"urn:schemas-microsoft-com:office:office\" lang=\"id\">"
                + "<head>"
                + "<meta charset=\"UTF-8\">"
                + "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">"
                + "<meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge\">"
                + "<meta name=\"color-scheme\" content=\"light\">"
                + "<meta name=\"supported-color-schemes\" content=\"light\">"
                + "<link rel=\"preconnect\" href=\"https://fonts.googleapis.com\">"
                + "<link href=\"" + FONT_IMPORT + "\" rel=\"stylesheet\">"
                + "<title>Konfirmasi Booking — Elvove</title>"
                + "<!--[if mso]><noscript><xml><o:OfficeDocumentSettings>"
                + "<o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->"
                + "</head>"
                + "<body style=\"margin:0;padding:0;width:100%;background-color:" + PAGE + ";"
                + "-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;\">"
                // Hidden preview text shown in the inbox list.
                + "<div style=\"display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;"
                + "mso-hide:all;font-size:1px;line-height:1px;\">" + escape(preheader) + "</div>"
                + "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;background-color:" + PAGE + ";border-collapse:collapse;\">"
                + "<tr><td align=\"center\" style=\"padding:32px 12px 44px 12px;\">"
                + "<!--[if mso]><table role=\"presentation\" width=\"600\" cellpadding=\"0\" cellspacing=\"0\" "
                + "border=\"0\"><tr><td><![endif]-->"
                + "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;max-width:600px;margin:0 auto;border-collapse:collapse;\">"
                + headerBlock()
                + contentBlock(body)
                + "</table>"
                + footerBlock()
                + "<!--[if mso]></td></tr></table><![endif]-->"
                + "</td></tr></table>"
                + "</body></html>";
    }

    /**
     * White header: the logo on the left, with the brand name in the brand red
     * beside it. The logo is a fixed square, so it is never stretched.
     */
    private String headerBlock() {
        return "<tr><td style=\"background:" + CARD + ";border:1px solid " + BORDER + ";border-bottom:1px solid "
                + BORDER + ";border-radius:12px 12px 0 0;padding:22px 32px;font-family:" + FONT + ";\">"
                + "<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"border-collapse:collapse;\"><tr>"
                + "<td valign=\"middle\" style=\"vertical-align:middle;padding:0 12px 0 0;width:44px;\">"
                // Decorative: the brand name beside it is the readable text, so
                // a blocked image leaves no clipped alt text behind.
                + "<img src=\"" + escape(logoUrl()) + "\" width=\"44\" height=\"44\" alt=\"\" "
                + "style=\"display:block;width:44px;height:44px;border:0;outline:none;text-decoration:none;\">"
                + "</td>"
                + "<td valign=\"middle\" style=\"vertical-align:middle;font-family:" + FONT + ";font-size:22px;"
                + "line-height:44px;font-weight:700;letter-spacing:2px;color:" + BRAND_RED + ";\">ELVOVE</td>"
                + "</tr></table>"
                + "</td></tr>";
    }

    /** The public logo on the site; email clients load it over HTTPS. */
    private String logoUrl() {
        return siteUrl.replaceAll("/+$", "") + "/logo.png";
    }

    /** White body of the card. */
    private String contentBlock(String body) {
        return "<tr><td style=\"background:" + CARD + ";border:1px solid " + BORDER + ";border-top:none;"
                + "border-radius:0 0 12px 12px;padding:34px 34px 38px 34px;font-family:" + FONT + ";\">"
                + body
                + "</td></tr>";
    }

    /** Centred status badge. */
    private String badge(String text) {
        return "<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" align=\"center\" "
                + "style=\"margin:0 auto;\"><tr>"
                + "<td style=\"background:" + BADGE_BG + ";border-radius:999px;padding:8px 14px;"
                + "font-family:" + FONT + ";font-size:10px;line-height:14px;font-weight:700;letter-spacing:1.5px;"
                + "color:" + BADGE_TEXT + ";text-align:center;white-space:nowrap;\">"
                + escape(text)
                + "</td></tr></table>";
    }

    /** Centred headline with its supporting line. */
    private String headingBlock(String headline, String description) {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td align=\"center\" style=\"padding:26px 0 0 0;font-family:" + FONT + ";font-size:25px;"
                + "line-height:35px;font-weight:400;color:" + TEXT + ";text-align:center;\">"
                + escape(headline)
                + "</td></tr>"
                + "<tr><td align=\"center\" style=\"padding:12px 0 0 0;font-family:" + FONT + ";font-size:14px;"
                + "line-height:23px;color:" + MUTED + ";text-align:center;\">"
                + escape(description)
                + "</td></tr>"
                + "</table>";
    }

    /** Left-aligned, personalised greeting. */
    private String greetingBlock(String greeting, String paragraph) {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td style=\"padding:30px 0 0 0;font-family:" + FONT + ";font-size:14px;line-height:24px;"
                + "color:" + TEXT + ";\">"
                + escape(greeting)
                + "</td></tr>"
                + "<tr><td style=\"padding:8px 0 0 0;font-family:" + FONT + ";font-size:14px;line-height:24px;"
                + "color:" + MUTED + ";\">"
                + escape(paragraph)
                + "</td></tr>"
                + "</table>";
    }

    /** The booking card: a bordered white panel holding a two-column table. */
    private String detailsCard(Booking booking) {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td style=\"padding:30px 0 0 0;\">"
                + "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border:1px solid " + BORDER + ";border-radius:10px;\">"
                + "<tr><td colspan=\"2\" style=\"padding:18px 18px 4px 18px;font-family:" + FONT + ";font-size:11px;"
                + "line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + MUTED + ";\">"
                + "DETAIL BOOKING"
                + "</td></tr>"
                + detailRow("Nama Pemesan", oneLine(booking.getName()), false, false)
                + detailRow("Tanggal Booking", tanggal(booking.getEventDate()), false, false)
                + detailRow("Waktu Sesi", jam(booking.getEventTime()), false, false)
                + detailRow("Lokasi", oneLine(booking.getEventLocation()), false, false)
                + detailRow("Paket", oneLine(booking.getPackageTitle()), false, false)
                + detailRow("Total Harga", rupiah(booking.getPackagePrice()), false, true)
                // The down payment and the remaining balance are agreed with the
                // client, so they are shown as pending rather than as a figure
                // the booking does not hold.
                + detailRow("DP (Uang Muka)", PENDING, false, true)
                + detailRow("Sisa Pembayaran", PENDING, true, true)
                + "</table>"
                + "</td></tr></table>";
    }

    /** The admin booking card: the booking ID and customer contact, then the event details. */
    private String adminDetailsCard(Booking booking, String id) {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border:1px solid " + BORDER + ";border-radius:10px;margin-top:30px;\">"
                + "<tr><td colspan=\"2\" style=\"padding:18px 18px 4px 18px;font-family:" + FONT + ";font-size:11px;"
                + "line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + MUTED + ";\">"
                + "DETAIL BOOKING"
                + "</td></tr>"
                + detailRow("ID Booking", "#" + id, false, false)
                + detailRow("Nama Pemesan", oneLine(booking.getName()), false, false)
                + detailRow("Email", oneLine(booking.getEmail()), false, false)
                + detailRow("Telepon", oneLine(booking.getPhone()), false, false)
                + detailRow("Tanggal Booking", tanggal(booking.getEventDate()), false, false)
                + detailRow("Waktu Sesi", jam(booking.getEventTime()), false, false)
                + detailRow("Lokasi", oneLine(booking.getEventLocation()), false, false)
                + detailRow("Paket", oneLine(booking.getPackageTitle()), false, false)
                + detailRow("Total Harga", rupiah(booking.getPackagePrice()), false, true)
                + detailRow("Status", statusLabel(booking), true, false)
                + "</table>";
    }

    /** Button to the booking in the admin panel. */
    private String adminButton(String link) {
        String href = escape(link);
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td align=\"center\" style=\"padding:26px 0 0 0;\">"
                + "<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">"
                + "<tr><td align=\"center\" bgcolor=\"" + BRAND_RED + "\" style=\"background:" + BRAND_RED
                + ";border-radius:5px;\">"
                + "<!--[if mso]>"
                + "<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" "
                + "xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" "
                + "style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" "
                + "fillcolor=\"" + BRAND_RED + "\">"
                + "<w:anchorlock/>"
                + "<center style=\"color:#ffffff;font-family:Arial,sans-serif;font-size:12px;font-weight:bold;"
                + "letter-spacing:1px;\">BUKA DI PANEL ADMIN</center>"
                + "</v:roundrect>"
                + "<![endif]-->"
                + "<!--[if !mso]><!-- -->"
                + "<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;"
                + "font-family:" + FONT + ";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;"
                + "color:#FFFFFF;text-decoration:none;\">BUKA DI PANEL ADMIN</a>"
                + "<!--<![endif]-->"
                + "</td></tr></table>"
                + "</td></tr></table>";
    }

    /** One row of the two-column table: the label on the left, the value on the right. */
    private String detailRow(String label, String value, boolean last, boolean price) {
        String border = last ? "" : "border-bottom:1px solid " + BORDER + ";";
        String valueColor = price ? ACCENT : TEXT;
        String valueWeight = price ? "600" : "400";
        // The closing row carries the card's bottom breathing room.
        String pad = last ? "padding:12px 18px 20px 18px;" : "padding:12px 18px;";
        return "<tr>"
                + "<td width=\"40%\" valign=\"top\" style=\"width:40%;" + pad + border
                + "font-family:" + FONT + ";font-size:13px;line-height:19px;color:" + MUTED + ";\">"
                + escape(label)
                + "</td>"
                + "<td valign=\"top\" style=\"" + pad + border + "font-family:" + FONT + ";font-size:13px;"
                + "line-height:19px;font-weight:" + valueWeight + ";color:" + valueColor + ";text-align:right;\">"
                + escape(value)
                + "</td>"
                + "</tr>";
    }

    /**
     * The highlighted next-steps box. The accent bar is a table cell rather
     * than a border, because Outlook drops borders on cells.
     */
    private String nextStepsBlock() {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td style=\"padding:26px 0 0 0;\">"
                + "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr>"
                + "<td width=\"3\" style=\"width:3px;background:" + ACCENT + ";border-radius:6px 0 0 6px;"
                + "font-size:0;line-height:0;\">&nbsp;</td>"
                + "<td style=\"background:" + INFO_BG + ";border-radius:0 6px 6px 0;padding:18px;\">"
                + "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td colspan=\"2\" style=\"padding:0 0 10px 0;font-family:" + FONT + ";font-size:11px;"
                + "line-height:16px;font-weight:700;letter-spacing:1.5px;color:" + ACCENT + ";\">"
                + "LANGKAH SELANJUTNYA"
                + "</td></tr>"
                + stepRow("Tim kami akan menghubungi Anda untuk mengonfirmasi down payment (DP) dan menyepakati "
                + "jadwal sesi foto.")
                + stepRow("Mohon siapkan konfirmasi mengenai paket, durasi, dan kebutuhan khusus untuk sesi Anda.")
                + stepRow("Datang tepat waktu di lokasi yang telah disepakati agar sesi berjalan optimal.")
                + stepRow("Punya pertanyaan atau perlu mengubah jadwal? Balas email ini, tim kami siap membantu.")
                + "</table>"
                + "</td>"
                + "</tr>"
                + "</table>"
                + "</td></tr></table>";
    }

    /** One bullet of the next-steps box. */
    private String stepRow(String step) {
        return "<tr>"
                + "<td width=\"14\" valign=\"top\" style=\"width:14px;padding:0 0 8px 0;font-family:" + FONT
                + ";font-size:13px;line-height:22px;color:" + ACCENT + ";\">&bull;</td>"
                + "<td valign=\"top\" style=\"padding:0 0 8px 0;font-family:" + FONT + ";font-size:13px;"
                + "line-height:22px;color:" + TEXT + ";\">"
                + escape(step)
                + "</td>"
                + "</tr>";
    }

    /** Reply button. The address is the one the application already uses. */
    private String contactButton(String contact) {
        // The sender address is configured in the application and is never
        // hardcoded here. If it is ever missing, the button points at the
        // site rather than at a broken "mailto:" with no address.
        String href = contact.isEmpty() ? escape(siteUrl) : "mailto:" + escape(contact);
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;border-collapse:collapse;\">"
                + "<tr><td align=\"center\" style=\"padding:26px 0 0 0;\">"
                + "<table role=\"presentation\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\">"
                + "<tr><td align=\"center\" bgcolor=\"" + DARK + "\" style=\"background:" + DARK
                + ";border-radius:5px;\">"
                + "<!--[if mso]>"
                + "<v:roundrect xmlns:v=\"urn:schemas-microsoft-com:vml\" "
                + "xmlns:w=\"urn:schemas-microsoft-com:office:word\" href=\"" + href + "\" "
                + "style=\"height:46px;v-text-anchor:middle;width:230px;\" arcsize=\"10%\" stroke=\"f\" "
                + "fillcolor=\"" + DARK + "\">"
                + "<w:anchorlock/>"
                + "<center style=\"color:#ffffff;font-family:Arial,sans-serif;font-size:12px;font-weight:bold;"
                + "letter-spacing:1px;\">BALAS EMAIL KAMI</center>"
                + "</v:roundrect>"
                + "<![endif]-->"
                + "<!--[if !mso]><!-- -->"
                + "<a href=\"" + href + "\" style=\"display:inline-block;padding:13px 24px;"
                + "font-family:" + FONT + ";font-size:12px;line-height:20px;font-weight:700;letter-spacing:1.2px;"
                + "color:#FFFFFF;text-decoration:none;\">BALAS EMAIL KAMI</a>"
                + "<!--<![endif]-->"
                + "</td></tr></table>"
                + "</td></tr></table>";
    }

    /** Minimalist centred footer. */
    private String footerBlock() {
        return "<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" "
                + "style=\"width:100%;max-width:600px;margin:0 auto;border-collapse:collapse;\">"
                + "<tr><td align=\"center\" style=\"padding:26px 16px 0 16px;font-family:" + FONT + ";\">"
                + "<div style=\"font-size:11px;line-height:16px;font-weight:700;letter-spacing:2px;color:" + MUTED
                + ";\">Elvove</div>"
                + "<div style=\"font-size:11px;line-height:18px;color:" + MUTED + ";padding-top:6px;\">"
                + TAGLINE
                + "</div>"
                + "<div style=\"font-size:10px;line-height:16px;color:" + MUTED + ";padding-top:6px;\">"
                + "© " + Year.now().getValue() + " " + BRAND_FALLBACK + ". All rights reserved."
                + "</div>"
                + "<div style=\"font-size:10px;line-height:16px;color:" + MUTED + ";padding-top:4px;\">"
                + "<a href=\"" + escape(siteUrl) + "\" style=\"color:" + MUTED + ";text-decoration:underline;\">"
                + escape(siteUrl)
                + "</a>"
                + "</div>"
                + "</td></tr></table>";
    }

    // -------------------------------------------------------------- details

    private String detailLines(Booking booking) {
        return "Nama Pemesan    : " + oneLine(booking.getName()) + "\n"
                + "Tanggal Booking : " + tanggal(booking.getEventDate()) + "\n"
                + "Waktu Sesi      : " + jam(booking.getEventTime()) + "\n"
                + "Lokasi          : " + oneLine(booking.getEventLocation()) + "\n"
                + "Paket           : " + oneLine(booking.getPackageTitle()) + "\n"
                + "Total Harga     : " + rupiah(booking.getPackagePrice()) + "\n"
                + "DP (Uang Muka)  : " + PENDING + "\n"
                + "Sisa Pembayaran : " + PENDING + "\n";
    }

    private String adminDetailLines(Booking booking, String id) {
        return "ID Booking        : #" + id + "\n"
                + "Nama Pemesan    : " + oneLine(booking.getName()) + "\n"
                + "Email           : " + oneLine(booking.getEmail()) + "\n"
                + "Telepon         : " + oneLine(booking.getPhone()) + "\n"
                + "Tanggal Booking : " + tanggal(booking.getEventDate()) + "\n"
                + "Waktu Sesi      : " + jam(booking.getEventTime()) + "\n"
                + "Lokasi          : " + oneLine(booking.getEventLocation()) + "\n"
                + "Paket           : " + oneLine(booking.getPackageTitle()) + "\n"
                + "Total Harga     : " + rupiah(booking.getPackagePrice()) + "\n"
                + "Status          : " + statusLabel(booking) + "\n";
    }

    private static String statusLabel(Booking booking) {
        if (booking.getStatus() == null) {
            return PENDING;
        }
        return switch (booking.getStatus()) {
            case PENDING -> "Menunggu";
            case ACCEPTED -> "Diterima";
            case RESCHEDULE_REQUESTED -> "Minta Ubah Jadwal";
            case COMPLETED -> "Selesai";
        };
    }

    private String stepsText() {
        return "• Tim kami akan menghubungi Anda untuk mengonfirmasi down payment (DP) dan menyepakati jadwal "
                + "sesi foto.\n"
                + "• Mohon siapkan konfirmasi mengenai paket, durasi, dan kebutuhan khusus untuk sesi Anda.\n"
                + "• Datang tepat waktu di lokasi yang telah disepakati agar sesi berjalan optimal.\n"
                + "• Punya pertanyaan atau perlu mengubah jadwal? Balas email ini, tim kami siap membantu.\n";
    }

    private String preview(Booking booking, String headline) {
        return headline + " — " + oneLine(booking.getName()) + ". "
                + "Sesi foto Anda pada " + tanggal(booking.getEventDate())
                + ". Terima kasih telah memilih Elvove.";
    }

    private String brandName(String senderName) {
        return senderName == null || senderName.isBlank() ? BRAND_FALLBACK : oneLine(senderName);
    }

    private String contactEmail(String senderEmail) {
        return senderEmail == null || senderEmail.isBlank() ? "" : oneLine(senderEmail);
    }

    private String tanggal(LocalDate date) {
        return date == null ? PENDING : date.format(TANGGAL);
    }

    private String jam(LocalTime time) {
        return time == null ? PENDING : time.format(JAM);
    }

    /** Indonesian Rupiah with thousand separators, for example {@code Rp200.000}. */
    private String rupiah(BigDecimal amount) {
        if (amount == null) {
            return PENDING;
        }
        DecimalFormat format = new DecimalFormat("#,##0", DecimalFormatSymbols.getInstance(ID));
        return "Rp" + format.format(amount);
    }

    // -------------------------------------------------------------- helpers

    /**
     * Escapes the characters that would otherwise be able to break out of the
     * text or an attribute context.
     */
    private static String escape(String value) {
        if (value == null) {
            return "";
        }
        StringBuilder out = new StringBuilder(value.length() + 16);
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            switch (c) {
                case '&' -> out.append("&amp;");
                case '<' -> out.append("&lt;");
                case '>' -> out.append("&gt;");
                case '"' -> out.append("&quot;");
                case '\'' -> out.append("&#39;");
                default -> out.append(c);
            }
        }
        return out.toString();
    }

    /** Collapses line breaks so a value cannot break the layout. */
    private static String oneLine(String value) {
        if (value == null) {
            return "";
        }
        return value.replaceAll("[\\r\\n\\t]+", " ").trim();
    }
}

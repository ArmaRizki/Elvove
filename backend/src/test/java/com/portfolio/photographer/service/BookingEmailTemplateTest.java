package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.autoconfigure.AutoConfigurations;
import org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration;
import org.springframework.boot.test.context.runner.ApplicationContextRunner;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Checks the rendered booking e-mail: the real booking data ends up in the
 * message, dynamic values cannot inject markup, prices and dates use the
 * Indonesian formats and the document stays inside the rules an email client
 * can render.
 */
class BookingEmailTemplateTest {

    private BookingEmailTemplate template;
    private Booking booking;

    @BeforeEach
    void setUp() {
        template = new BookingEmailTemplate("https://elvove.site");

        booking = new Booking();
        booking.setId(7L);
        booking.setName("Anindya Putri");
        booking.setEmail("anindya@example.com");
        booking.setEventDate(LocalDate.of(2026, 11, 10));
        booking.setEventTime(LocalTime.of(9, 0));
        booking.setEventLocation("The Grand Ballroom, Jakarta");
        booking.setPackageTitle("Premium Wedding Full Day");
        booking.setPackagePrice(new BigDecimal("18500000"));
    }

    @Test
    void confirmationShowsEveryBookingDetail() {
        var content = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com");

        assertThat(content.html())
                .contains("Anindya Putri")
                .doesNotContain("Wedding Reception")
                .contains("10 November 2026")
                .contains("09:00")
                .contains("The Grand Ballroom, Jakarta")
                .contains("Premium Wedding Full Day")
                .contains("Rp18.500.000");
    }

    @Test
    void confirmationContainsEveryRequiredSection() {
        var content = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com");
        String html = content.html();

        assertThat(html)
                // A. white header with the logo and brand name
                .contains("background:#FFFFFF")
                .contains("<img src=\"https://elvove.site/logo.png\"")
                .contains("letter-spacing:2px;color:#E11D48;\">ELVOVE</td>")
                // B. status badge and headline
                .contains("PESANAN DITERIMA")
                .contains("Booking Anda Telah Dikonfirmasi")
                // C. personalised greeting
                .contains("Halo, Anindya Putri.")
                .contains("Terima kasih telah memilih Elvove.")
                .contains("Berikut adalah detail reservasi Anda.")
                // D. booking details card
                .contains("DETAIL BOOKING")
                .contains("Nama Pemesan")
                .doesNotContain("Jenis Layanan")
                .contains("Tanggal Booking")
                .contains("Waktu Sesi")
                .contains("Lokasi")
                .contains("Paket")
                .contains("Total Harga")
                .contains("DP (Uang Muka)")
                .contains("Sisa Pembayaran")
                // E. next steps
                .contains("LANGKAH SELANJUTNYA")
                .contains("down payment (DP)")
                .contains("Balas email ini")
                // F. contact button
                .contains("BALAS EMAIL KAMI")
                .contains("mailto:elvo.vee@gmail.com")
                // G. footer
                .contains("Elvove</div>")
                .contains("Capturing Moments, Creating Memories.")
                .contains("Elvove. All rights reserved.");
    }

    @Test
    void theHeaderIsWhiteWithTheLogoOnTheLeftAndTheBrandRedName() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("background:#FFFFFF")
                .contains("vertical-align:middle")
                .contains("width=\"44\" height=\"44\"")
                .contains("width:44px;height:44px")
                .contains("font-size:22px")
                .contains("color:#E11D48")
                .doesNotContain("background:#171525;border-radius:12px 12px 0 0");
    }

    @Test
    void theLogoIsAPublicHttpsUrlAndNeverALocalPath() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("src=\"https://elvove.site/logo.png\"")
                .doesNotContain("file:")
                .doesNotContain("cid:")
                .doesNotContain("/home/");
    }

    @Test
    void theTypographyUsesInterWithSafeFallbacks() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap")
                .contains("font-family:Inter, Arial, Helvetica, sans-serif");
    }

    @Test
    void theBadgeAndTheHeadlineAreCentered() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("background:#FCEEF1")
                .contains("color:#E11D48")
                .contains("font-size:10px")
                .contains("letter-spacing:1.5px")
                .contains("padding:8px 14px")
                .contains("align=\"center\"")
                .contains("font-size:25px")
                .contains("line-height:35px");
    }

    @Test
    void theDetailsCardUsesATwoColumnTableWithSeparators() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("border:1px solid #EEEAF2")
                .contains("border-radius:10px")
                .contains("width=\"40%\"")
                .contains("font-size:13px")
                .contains("padding:12px 18px")
                .contains("border-bottom:1px solid #EEEAF2");
    }

    @Test
    void priceValuesUseTheAccentColor() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("color:#E11D48")
                .contains("font-weight:600");
    }

    @Test
    void downPaymentAndRemainingPaymentAreShownAsPending() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        // The booking holds no down-payment amount, so the card must not invent
        // one: the two money rows stay pending until the team agrees them.
        assertThat(html)
                .contains("DP (Uang Muka)")
                .contains("Sisa Pembayaran")
                .contains("Akan dikonfirmasi")
                .doesNotContain("DP (Uang Muka)  : Rp");
    }

    @Test
    void theNextStepsBoxUsesTheLavenderBackgroundAndAccentBar() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("background:#F4F1F8")
                .contains("width:3px;background:#E11D48")
                .contains("padding:18px")
                .contains("border-radius:0 6px 6px 0")
                .contains("LANGKAH SELANJUTNYA");
    }

    @Test
    void theButtonUsesTheConfiguredBusinessAddress() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("BALAS EMAIL KAMI")
                .contains("mailto:elvo.vee@gmail.com")
                .contains("background:#171525")
                .contains("padding:13px 24px")
                .contains("border-radius:5px");
    }

    @Test
    void theButtonFallsBackToTheSiteWhenNoSenderAddressIsConfigured() {
        String html = template.confirmation(booking, "Elvove", "  ").html();

        // No dead "mailto:" with an empty address: the button points at the site.
        assertThat(html)
                .contains("BALAS EMAIL KAMI")
                .doesNotContain("mailto:\"")
                .contains("href=\"https://elvove.site\"");
    }

    @Test
    void everyTextElementUsesTheSpecifiedFontFamily() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("font-family:Inter, Arial, Helvetica, sans-serif")
                .doesNotContain("font-family:Inter,Arial,Helvetica,sans-serif");
    }

    @Test
    void theFooterIsMinimalAndCentred() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("Elvove</div>")
                .contains("Capturing Moments, Creating Memories.")
                .contains("All rights reserved.")
                .contains("color:#858191");
    }

    @Test
    void priceIsFormattedAsIndonesianRupiah() {
        booking.setPackagePrice(new BigDecimal("200000"));
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html())
                .contains("Rp200.000");

        booking.setPackagePrice(new BigDecimal("1250000"));
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html())
                .contains("Rp1.250.000");

        booking.setPackagePrice(new BigDecimal("18500000"));
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html())
                .contains("Rp18.500.000");
    }

    @Test
    void dateIsFormattedInAReadableIndonesianFormat() {
        booking.setEventDate(LocalDate.of(2026, 12, 1));
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html())
                .contains("1 Desember 2026");

        booking.setEventDate(LocalDate.of(2026, 3, 5));
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html())
                .contains("5 Maret 2026");
    }

    @Test
    void specialCharactersInCustomerDataAreEscaped() {
        booking.setName("<script>alert(\"x\")</script>");
        booking.setEventLocation("Hotel & Resort <b>\"Istana\"</b>");

        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .doesNotContain("<script>")
                .doesNotContain("<b>")
                .contains("&lt;script&gt;")
                .contains("Hotel &amp; Resort &lt;b&gt;&quot;Istana&quot;&lt;/b&gt;");
    }

    @Test
    void quotesInTheSenderAddressCannotBreakOutOfTheMailtoLink() {
        String html = template.confirmation(booking, "Elvove", "a\"b@example.com").html();

        assertThat(html)
                .doesNotContain("mailto:a\"b@example.com")
                .contains("mailto:a&quot;b@example.com");
    }

    @Test
    void lineBreaksInCustomerDataCannotBreakTheLayout() {
        booking.setEventLocation("Gedung\nA\nB");

        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html).contains("Gedung A B");
    }

    @Test
    void documentIsTableBasedAndUsesNoUnsupportedFeatures() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .contains("<table")
                .contains("role=\"presentation\"")
                .contains("max-width:600px")
                .contains("width=\"100%\"")
                .doesNotContain("<script")
                .doesNotContain("<style")
                // The only <link> is the optional Inter stylesheet from Google Fonts.
                .doesNotContain("<link href=\"https://elvove.site")
                .doesNotContain("<style")
                .doesNotContain("display:flex")
                .doesNotContain("display:grid")
                .doesNotContain("@media")
                .doesNotContain("position:absolute");
    }

    @Test
    void missingOptionalDetailsFallBackToAReadablePlaceholder() {
        booking.setEventTime(null);
        booking.setEventLocation("   ");
        booking.setPackageTitle(null);
        booking.setPackagePrice(null);

        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html).contains("Akan dikonfirmasi");
    }

    @Test
    void plainTextPartCarriesTheSameDetails() {
        var content = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com");

        assertThat(content.text())
                .contains("PESANAN DITERIMA")
                .contains("Booking Anda Telah Dikonfirmasi")
                .contains("Halo, Anindya Putri.")
                .contains("DETAIL BOOKING")
                .contains("Nama Pemesan    : Anindya Putri")
                .doesNotContain("Jenis Layanan")
                .contains("Tanggal Booking : 10 November 2026")
                .contains("Waktu Sesi      : 09:00")
                .contains("Lokasi          : The Grand Ballroom, Jakarta")
                .contains("Paket           : Premium Wedding Full Day")
                .contains("Total Harga     : Rp18.500.000")
                .contains("DP (Uang Muka)  : Akan dikonfirmasi")
                .contains("Sisa Pembayaran : Akan dikonfirmasi")
                .contains("LANGKAH SELANJUTNYA")
                .contains("down payment (DP)")
                .contains("Capturing Moments, Creating Memories.");
    }

    @Test
    void theAdminNotificationNoLongerShowsTheServiceTypeRow() {
        var content = template.adminNotification(booking, "Elvove", "https://elvove.site/admin/bookings");

        assertThat(content.html())
                .contains("DETAIL BOOKING")
                .doesNotContain("Jenis Layanan")
                .doesNotContain("Wedding Reception");
        assertThat(content.text())
                .contains("DETAIL BOOKING")
                .doesNotContain("Jenis Layanan")
                .doesNotContain("Wedding Reception");
    }

    @Test
    void subjectNamesTheBrandAndThePurpose() {
        assertThat(template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").subject())
                .contains("Booking Anda Telah Dikonfirmasi")
                .contains("Elvove");
    }

    @Test
    void theMessageNeverCarriesInternalMailSettings() {
        String html = template.confirmation(booking, "Elvove", "elvo.vee@gmail.com").html();

        assertThat(html)
                .doesNotContain("smtp.")
                .doesNotContain("password")
                .doesNotContain("encrypted")
                .doesNotContain("SMTP_ENCRYPTION_KEY")
                .doesNotContain("STARTTLS");
    }

    @Test
    void rescheduleUsesTheSameLayoutAndCarriesTheNewSchedule() {
        booking.setEventDate(LocalDate.of(2026, 12, 5));
        booking.setEventTime(LocalTime.of(10, 0));

        var content = template.reschedule(booking, "Elvove", "elvo.vee@gmail.com");

        assertThat(content.html())
                .contains("background:#FFFFFF")
                .contains("PERUBAHAN JADWAL")
                .contains("Jadwal Sesi Foto Diperbarui")
                .contains("Halo, Anindya Putri.")
                .contains("5 Desember 2026")
                .contains("10:00")
                .contains("DETAIL BOOKING")
                .contains("LANGKAH SELANJUTNYA")
                .contains("BALAS EMAIL KAMI")
                .contains("mailto:elvo.vee@gmail.com")
                .contains("Capturing Moments, Creating Memories.");
        assertThat(content.text()).contains("5 Desember 2026").contains("10:00");
    }

    @Test
    void aBlankSenderNameFallsBackToTheStudioName() {
        var content = template.confirmation(booking, "  ", "elvo.vee@gmail.com");

        assertThat(content.subject()).contains("Elvove");
        assertThat(content.html()).contains("Elvove");
    }

    @Test
    void theTemplateBeanStartsWithTheConfiguredAssetDefaults() {
        new ApplicationContextRunner()
                .withConfiguration(AutoConfigurations.of(PropertyPlaceholderAutoConfiguration.class))
                .withBean(BookingEmailTemplate.class)
                .run(context -> {
                    assertThat(context).hasNotFailed();
                    assertThat(context).hasSingleBean(BookingEmailTemplate.class);
                });
    }
}

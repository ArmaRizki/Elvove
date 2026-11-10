package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;

import java.math.BigDecimal;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.LocalDate;
import java.time.LocalTime;

/**
 * Local preview of the booking e-mail. It renders the real template with
 * sample data and writes the result to an HTML file, so the design can be
 * reviewed in a browser without configuring SMTP and without sending anything
 * to a customer.
 *
 * <pre>
 * cd backend
 * mvn -o -q test-compile dependency:build-classpath -Dmdep.outputFile=target/cp.txt
 * java -cp "target/classes:target/test-classes:$(cat target/cp.txt)" \
 *      com.portfolio.photographer.service.BookingEmailPreview target/booking-preview.html
 * </pre>
 *
 * <p>Output defaults to {@code target/booking-preview.html}, which git ignores,
 * so running it never adds scratch files to the repository.</p>
 */
public final class BookingEmailPreview {

    private BookingEmailPreview() {}

    public static void main(String[] args) throws Exception {
        Path target = Path.of(args.length > 0 ? args[0] : "target/booking-preview.html");

        BookingEmailTemplate template = new BookingEmailTemplate("https://elvove.site");

        BookingEmailTemplate.EmailContent confirmation = template.confirmation(sampleBooking(),
                "Elvove", "elvo.vee@gmail.com");
        Files.writeString(target, confirmation.html(), StandardCharsets.UTF_8);

        Path text = target.resolveSibling(target.getFileName() + ".txt");
        Files.writeString(text, confirmation.text(), StandardCharsets.UTF_8);

        System.out.println("Subject : " + confirmation.subject());
        System.out.println("HTML    : " + target.toAbsolutePath());
        System.out.println("Plain   : " + text.toAbsolutePath());
    }

    private static Booking sampleBooking() {
        Booking booking = new Booking();
        booking.setId(1L);
        booking.setName("Arma Rizki");
        booking.setEmail("arma.rizki@example.com");
        booking.setPhone("+62 812 3456 7890");
        booking.setEventDate(LocalDate.now().plusDays(30));
        booking.setEventTime(LocalTime.of(9, 30));
        booking.setEventLocation("Villa Sentul, Bogor");
        booking.setPackageTitle("Signature Prewedding");
        booking.setPackagePrice(new BigDecimal("2500000"));
        return booking;
    }
}

package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import java.time.LocalDate;
import java.util.List;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BookingNotificationMailServiceTest {

    private static Booking booking() {
        Booking b = new Booking();
        b.setId(5L);
        b.setName("Dewi");
        b.setEmail("dewi@example.com");
        b.setPhone("08123");
        b.setEventDate(LocalDate.of(2026, 12, 1));
        b.setStatus(BookingStatus.PENDING);
        return b;
    }

    private static BookingNotificationMailService service(SmtpMailService smtp) {
        BookingNotificationMailService service = new BookingNotificationMailService(
                smtp, new BookingEmailTemplate("https://elvove.site"));
        ReflectionTestUtils.setField(service, "adminUrl", "https://elvove.site");
        return service;
    }

    @Test
    void oneRecipientFailureDoesNotStopTheOthers() {
        SmtpMailService smtp = mock(SmtpMailService.class);
        when(smtp.getSenderName()).thenReturn("Elvove");
        doThrow(new IllegalStateException("550 mailbox unavailable"))
                .when(smtp).sendHtml(eq("gagal@contoh.com"), anyString(), anyString(), anyString());

        service(smtp).sendBookingNotification(booking(),
                List.of("gagal@contoh.com", "ok@contoh.com", "ok2@contoh.com"));

        verify(smtp, times(1)).sendHtml(eq("gagal@contoh.com"), anyString(), anyString(), anyString());
        verify(smtp, times(1)).sendHtml(eq("ok@contoh.com"), anyString(), anyString(), anyString());
        verify(smtp, times(1)).sendHtml(eq("ok2@contoh.com"), anyString(), anyString(), anyString());
    }

    @Test
    void notificationContainsDeepLinkAndNoSecrets() {
        SmtpMailService smtp = mock(SmtpMailService.class);
        when(smtp.getSenderName()).thenReturn("Elvove");

        service(smtp).sendBookingNotification(booking(), List.of("admin@contoh.com"));

        verify(smtp).sendHtml(eq("admin@contoh.com"), anyString(),
                org.mockito.ArgumentMatchers.contains("https://elvove.site/admin/bookings?booking=5"),
                org.mockito.ArgumentMatchers.contains("https://elvove.site/admin/bookings?booking=5"));
    }
}

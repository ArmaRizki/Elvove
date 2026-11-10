package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.service.BookingNotificationSettingsService.Snapshot;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThatCode;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BookingNotificationWorkerTest {

    private static Booking booking() {
        Booking b = new Booking();
        b.setId(9L);
        b.setName("Raka");
        b.setEmail("raka@example.com");
        b.setEventDate(LocalDate.of(2026, 12, 20));
        b.setStatus(BookingStatus.PENDING);
        return b;
    }

    @Test
    void disabledSendsNothing() {
        BookingNotificationSettingsService settings = mock(BookingNotificationSettingsService.class);
        BookingNotificationMailService mail = mock(BookingNotificationMailService.class);
        when(settings.current()).thenReturn(new Snapshot(false, List.of("admin@contoh.com")));

        new BookingNotificationWorker(settings, mail).process(booking());

        verify(mail, never()).sendBookingNotification(any(), anyList());
    }

    @Test
    void enabledWithEmptyListWarnsAndSendsNothing() {
        BookingNotificationSettingsService settings = mock(BookingNotificationSettingsService.class);
        BookingNotificationMailService mail = mock(BookingNotificationMailService.class);
        when(settings.current()).thenReturn(new Snapshot(true, List.of()));

        new BookingNotificationWorker(settings, mail).process(booking());

        verify(mail, never()).sendBookingNotification(any(), anyList());
    }

    @Test
    void enabledWithRecipientsSendsToThem() {
        BookingNotificationSettingsService settings = mock(BookingNotificationSettingsService.class);
        BookingNotificationMailService mail = mock(BookingNotificationMailService.class);
        List<String> recipients = List.of("a@contoh.com", "b@contoh.com");
        when(settings.current()).thenReturn(new Snapshot(true, recipients));

        new BookingNotificationWorker(settings, mail).process(booking());

        verify(mail).sendBookingNotification(any(), any());
    }

    @Test
    void settingsFailureIsContained() {
        BookingNotificationSettingsService settings = mock(BookingNotificationSettingsService.class);
        BookingNotificationMailService mail = mock(BookingNotificationMailService.class);
        when(settings.current()).thenThrow(new IllegalStateException("db down"));

        assertThatCode(() -> new BookingNotificationWorker(settings, mail).process(booking()))
                .doesNotThrowAnyException();
    }
}

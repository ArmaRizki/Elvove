package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.service.BookingNotificationSettingsService.Snapshot;
import lombok.RequiredArgsConstructor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

/**
 * Background part of the booking notification. It lives in its own bean so the
 * call from the dispatcher goes through the async proxy and the executor.
 */
@Service
@RequiredArgsConstructor
public class BookingNotificationWorker {
    private static final Logger log = LoggerFactory.getLogger(BookingNotificationWorker.class);

    private final BookingNotificationSettingsService settingsService;
    private final BookingNotificationMailService mailService;

    @Async("bookingNotificationExecutor")
    public void process(Booking booking) {
        try {
            Snapshot settings = settingsService.current();
            if (!settings.enabled()) {
                return;
            }
            if (settings.recipients().isEmpty()) {
                log.warn("Notifikasi booking #{} dilewati: notifikasi aktif tetapi daftar penerima kosong.",
                        booking.getId());
                return;
            }
            mailService.sendBookingNotification(booking, settings.recipients());
        } catch (Exception e) {
            log.error("Gagal memproses notifikasi booking #{} ({})",
                    booking.getId(), e.getClass().getSimpleName());
        }
    }
}

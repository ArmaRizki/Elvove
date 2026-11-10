package com.portfolio.photographer.service;

import com.portfolio.photographer.event.BookingCreatedEvent;
import lombok.RequiredArgsConstructor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.task.TaskRejectedException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.event.TransactionPhase;
import org.springframework.transaction.event.TransactionalEventListener;

/**
 * Runs the administrator notification after the booking transaction commits.
 * The listener only submits work to the executor, so SMTP never delays the
 * customer or rolls back the booking. Each booking is submitted once, and a
 * rejected submission is logged with the booking ID, never dropped silently.
 */
@Service
@RequiredArgsConstructor
public class BookingNotificationDispatcher {
    private static final Logger log = LoggerFactory.getLogger(BookingNotificationDispatcher.class);

    private final BookingNotificationWorker worker;

    @TransactionalEventListener(phase = TransactionPhase.AFTER_COMMIT)
    public void onBookingCreated(BookingCreatedEvent event) {
        try {
            worker.process(event.booking());
        } catch (TaskRejectedException e) {
            log.error("Notifikasi booking #{} ditolak: antrean penuh. Tidak ada email yang dikirim untuk pemesanan ini.",
                    event.booking().getId());
        }
    }
}

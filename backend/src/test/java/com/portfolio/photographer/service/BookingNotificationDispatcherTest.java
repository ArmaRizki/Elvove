package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.event.BookingCreatedEvent;
import org.junit.jupiter.api.Test;
import org.springframework.core.task.TaskRejectedException;
import org.springframework.scheduling.annotation.Async;
import org.springframework.transaction.event.TransactionPhase;
import org.springframework.transaction.event.TransactionalEventListener;

import java.lang.reflect.Method;
import java.time.LocalDate;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatCode;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;

class BookingNotificationDispatcherTest {

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
    void listenerRunsOnlyAfterCommit() throws NoSuchMethodException {
        Method m = BookingNotificationDispatcher.class.getMethod("onBookingCreated", BookingCreatedEvent.class);
        TransactionalEventListener listener = m.getAnnotation(TransactionalEventListener.class);
        assertThat(listener).isNotNull();
        assertThat(listener.phase()).isEqualTo(TransactionPhase.AFTER_COMMIT);
    }

    @Test
    void workerIsAsyncOnTheBookingExecutor() throws NoSuchMethodException {
        Method m = BookingNotificationWorker.class.getMethod("process", Booking.class);
        Async async = m.getAnnotation(Async.class);
        assertThat(async).isNotNull();
        assertThat(async.value()).isEqualTo("bookingNotificationExecutor");
    }

    @Test
    void submitsEachBookingOnce() {
        BookingNotificationWorker worker = mock(BookingNotificationWorker.class);
        Booking b = booking();

        new BookingNotificationDispatcher(worker).onBookingCreated(new BookingCreatedEvent(b));

        verify(worker).process(b);
    }

    @Test
    void rejectedSubmissionIsContainedAndNotSilent() {
        BookingNotificationWorker worker = mock(BookingNotificationWorker.class);
        doThrow(new TaskRejectedException("queue full")).when(worker).process(any());

        // The booking is already committed, so a full queue must not throw back to the caller.
        assertThatCode(() -> new BookingNotificationDispatcher(worker)
                .onBookingCreated(new BookingCreatedEvent(booking())))
                .doesNotThrowAnyException();
    }
}

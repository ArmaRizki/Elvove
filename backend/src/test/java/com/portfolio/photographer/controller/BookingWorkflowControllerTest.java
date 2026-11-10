package com.portfolio.photographer.controller;

import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.repository.PackageItemRepository;
import com.portfolio.photographer.service.BookingEmailTemplate;
import com.portfolio.photographer.service.BookingService;
import com.portfolio.photographer.service.SmtpMailService;
import com.portfolio.photographer.service.TermsService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BookingWorkflowControllerTest {

    private BookingRepository repository;
    private SmtpMailService mailService;
    private BookingService bookingService;
    private BookingWorkflowController controller;
    private Booking booking;

    @BeforeEach
    void setUp() {
        repository = mock(BookingRepository.class);
        mailService = mock(SmtpMailService.class);
        bookingService = new BookingService(
                repository,
                mock(PackageItemRepository.class),
                mock(TermsService.class),
                mock(org.springframework.context.ApplicationEventPublisher.class));
        controller = new BookingWorkflowController(repository, bookingService, mailService,
                new BookingEmailTemplate("https://elvove.site"));

        booking = new Booking();
        booking.setId(7L);
        booking.setName("Anindya Putri");
        booking.setEmail("anindya@example.com");
        booking.setPhone("+62 812 3456 7890");
        booking.setEventDate(LocalDate.of(2026, 11, 10));
        booking.setEventTime(LocalTime.of(9, 0));
        booking.setEventLocation("The Grand Ballroom, Jakarta");
        booking.setPackageTitle("Premium Wedding Full Day");
        booking.setPackagePrice(new BigDecimal("18500000"));
        booking.setStatus(BookingStatus.PENDING);

        when(repository.findById(7L)).thenReturn(Optional.of(booking));
        when(repository.save(any(Booking.class))).thenAnswer(call -> call.getArgument(0));
        when(mailService.getSenderName()).thenReturn("Elvove");
        when(mailService.getSenderEmail()).thenReturn("elvo.vee@gmail.com");
    }

    @Test
    void acceptingBookingNeverSendsEmail() {
        var response = controller.accept(7L);

        assertThat(booking.getStatus()).isEqualTo(BookingStatus.ACCEPTED);
        verify(repository).save(booking);
        verify(mailService, never()).sendHtml(anyString(), anyString(), anyString(), anyString());
        assertThat(response.getBody()).containsEntry("status", BookingStatus.ACCEPTED);
        assertThat(booking.getConfirmationEmailSentAt()).isNull();
        assertThat(booking.getConfirmationEmailCount()).isZero();
    }

    @Test
    void acceptRejectsBookingThatIsAlreadyAccepted() {
        booking.setStatus(BookingStatus.ACCEPTED);

        assertThatThrownBy(() -> controller.accept(7L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("sudah berstatus");
        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void confirmationEmailRejectedForPendingBooking() {
        assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("sudah diterima");

        verify(mailService, never()).sendHtml(anyString(), anyString(), anyString(), anyString());
        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void confirmationEmailRejectedForEveryNonAcceptedStatus() {
        for (BookingStatus status : new BookingStatus[]{
                BookingStatus.PENDING,
                BookingStatus.RESCHEDULE_REQUESTED,
                BookingStatus.COMPLETED}) {
            booking.setStatus(status);

            assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                    .as("status %s", status)
                    .isInstanceOf(BadRequestException.class)
                    .hasMessageContaining("sudah diterima");
        }

        verify(mailService, never()).sendHtml(anyString(), anyString(), anyString(), anyString());
        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void confirmationEmailGoesToClientWithPackageDetails() {
        booking.setStatus(BookingStatus.ACCEPTED);

        var response = controller.sendConfirmationEmail(7L);

        ArgumentCaptor<String> html = ArgumentCaptor.forClass(String.class);
        ArgumentCaptor<String> text = ArgumentCaptor.forClass(String.class);
        verify(mailService).sendHtml(eq("anindya@example.com"), anyString(), html.capture(), text.capture());

        assertThat(html.getValue())
                .contains("Anindya Putri")
                .contains("10 November 2026")
                .contains("09:00")
                .contains("Premium Wedding Full Day")
                .contains("Rp18.500.000")
                .contains("The Grand Ballroom, Jakarta")
                .contains("PESANAN DITERIMA")
                .contains("Booking Anda Telah Dikonfirmasi");
        assertThat(text.getValue())
                .contains("Anindya Putri")
                .contains("10 November 2026")
                .contains("09:00")
                .contains("Premium Wedding Full Day")
                .contains("Rp18.500.000")
                .contains("The Grand Ballroom, Jakarta")
                .contains("PESANAN DITERIMA")
                .contains("Booking Anda Telah Dikonfirmasi");
        assertThat(html.getValue()).doesNotContain("smtp.gmail.com", "password");
        assertThat(text.getValue()).doesNotContain("smtp.gmail.com", "password");
        assertThat(response.getBody()).containsEntry("sentTo", "anindya@example.com");
    }

    @Test
    void successfulSendPersistsTrackingAndReturnsIt() {
        booking.setStatus(BookingStatus.ACCEPTED);

        var response = controller.sendConfirmationEmail(7L);

        assertThat(booking.getConfirmationEmailSentAt()).isNotNull();
        assertThat(booking.getConfirmationEmailCount()).isEqualTo(1);
        assertThat(response.getBody())
                .containsEntry("confirmationEmailCount", 1)
                .containsKey("confirmationEmailSentAt");
        verify(repository).save(booking);
    }

    @Test
    void failedDeliveryLeavesTrackingUntouchedAndShowsNoSmtpDetail() {
        booking.setStatus(BookingStatus.ACCEPTED);
        doThrow(new IllegalStateException("Gagal mengirim email SMTP: connection refused"))
                .when(mailService).sendHtml(anyString(), anyString(), anyString(), anyString());

        assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("gagal dikirim")
                .hasMessageNotContaining("connection refused");

        assertThat(booking.getConfirmationEmailSentAt()).isNull();
        assertThat(booking.getConfirmationEmailCount()).isZero();
        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void retryAfterFailureSendsAgainAndOnlyThenRecordsTracking() {
        booking.setStatus(BookingStatus.ACCEPTED);
        doThrow(new IllegalStateException("SMTP down"))
                .doNothing()
                .when(mailService).sendHtml(anyString(), anyString(), anyString(), anyString());

        assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                .isInstanceOf(BadRequestException.class);
        assertThat(booking.getConfirmationEmailSentAt()).isNull();
        assertThat(booking.getConfirmationEmailCount()).isZero();

        // The retry goes through: the in-flight guard must not stay locked.
        var response = controller.sendConfirmationEmail(7L);

        assertThat(response.getBody()).containsEntry("sentTo", "anindya@example.com");
        assertThat(booking.getConfirmationEmailSentAt()).isNotNull();
        assertThat(booking.getConfirmationEmailCount()).isEqualTo(1);
        verify(mailService, times(2)).sendHtml(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void secondSendIncrementsCountAndRefreshesTimestamp() {
        booking.setStatus(BookingStatus.ACCEPTED);

        controller.sendConfirmationEmail(7L);
        var second = controller.sendConfirmationEmail(7L);

        assertThat(booking.getConfirmationEmailCount()).isEqualTo(2);
        assertThat(second.getBody()).containsEntry("confirmationEmailCount", 2);
    }

    @Test
    void concurrentDuplicateRequestIsRejectedWhileFirstIsInFlight() throws Exception {
        booking.setStatus(BookingStatus.ACCEPTED);

        CountDownLatch enteredSmtp = new CountDownLatch(1);
        CountDownLatch releaseSmtp = new CountDownLatch(1);
        doAnswer(call -> {
            enteredSmtp.countDown();
            releaseSmtp.await(5, TimeUnit.SECONDS);
            return null;
        }).when(mailService).sendHtml(anyString(), anyString(), anyString(), anyString());

        ExecutorService executor = Executors.newSingleThreadExecutor();
        try {
            Future<?> first = executor.submit(() -> controller.sendConfirmationEmail(7L));
            assertThat(enteredSmtp.await(5, TimeUnit.SECONDS)).isTrue();

            assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                    .isInstanceOf(BadRequestException.class)
                    .hasMessageContaining("sedang dikirim");

            releaseSmtp.countDown();
            first.get(5, TimeUnit.SECONDS);
        } finally {
            releaseSmtp.countDown();
            executor.shutdownNow();
        }

        assertThat(booking.getConfirmationEmailCount()).isEqualTo(1);
        verify(mailService, times(1)).sendHtml(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void confirmationEmailRequiresClientAddress() {
        booking.setStatus(BookingStatus.ACCEPTED);
        booking.setEmail("   ");

        assertThatThrownBy(() -> controller.sendConfirmationEmail(7L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("tidak memiliki alamat email");

        verify(mailService, never()).sendHtml(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void confirmationEmailForUnknownBookingIsNotFound() {
        when(repository.findById(99L)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> controller.sendConfirmationEmail(99L))
                .isInstanceOf(ResourceNotFoundException.class)
                .hasMessageContaining("Pemesanan tidak ditemukan");
    }

    @Test
    void rescheduleFromPendingIsAllowedAndNotifiesClient() {
        var request = new BookingWorkflowController.RescheduleRequest(
                LocalDate.of(2026, 12, 1), LocalTime.of(13, 30));

        var response = controller.reschedule(7L, request);

        assertThat(booking.getStatus()).isEqualTo(BookingStatus.RESCHEDULE_REQUESTED);
        assertThat(booking.getEventDate()).isEqualTo(LocalDate.of(2026, 12, 1));
        assertThat(booking.getEventTime()).isEqualTo(LocalTime.of(13, 30));
        assertThat(response.getBody()).containsEntry("status", BookingStatus.RESCHEDULE_REQUESTED);
        verify(mailService).sendHtml(eq("anindya@example.com"), anyString(), anyString(), anyString());
    }

    @Test
    void rescheduleFromCompletedIsRejectedWithoutSendingEmail() {
        booking.setStatus(BookingStatus.COMPLETED);
        var request = new BookingWorkflowController.RescheduleRequest(
                LocalDate.of(2026, 12, 1), LocalTime.of(13, 30));

        assertThatThrownBy(() -> controller.reschedule(7L, request))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("tidak dapat diubah");

        assertThat(booking.getEventDate()).isEqualTo(LocalDate.of(2026, 11, 10));
        verify(mailService, never()).sendHtml(anyString(), anyString(), anyString(), anyString());
        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void secondScheduleAdjustmentWhileAwaitingRescheduleIsAllowed() {
        booking.setStatus(BookingStatus.RESCHEDULE_REQUESTED);
        var request = new BookingWorkflowController.RescheduleRequest(
                LocalDate.of(2026, 12, 5), LocalTime.of(10, 0));

        var response = controller.reschedule(7L, request);

        assertThat(booking.getStatus()).isEqualTo(BookingStatus.RESCHEDULE_REQUESTED);
        assertThat(response.getBody()).containsEntry("status", BookingStatus.RESCHEDULE_REQUESTED);
        verify(mailService).sendHtml(eq("anindya@example.com"), anyString(), anyString(), anyString());
    }

    @Test
    void trackingWriteFailureIsReportedSeparatelyAfterSuccessfulSmtp() {
        booking.setStatus(BookingStatus.ACCEPTED);
        BookingService failingTracking = mock(BookingService.class);
        when(failingTracking.recordConfirmationEmailSent(7L))
                .thenThrow(new IllegalStateException("database down"));
        BookingWorkflowController withFailingTracking =
                new BookingWorkflowController(repository, failingTracking, mailService,
                        new BookingEmailTemplate("https://elvove.site"));

        assertThatThrownBy(() -> withFailingTracking.sendConfirmationEmail(7L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("status pengiriman gagal disimpan");

        verify(mailService).sendHtml(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void legacyResendEndpointNoLongerExists() {
        boolean hasResendMapping = java.util.Arrays.stream(
                        BookingWorkflowController.class.getDeclaredMethods())
                .flatMap(method -> java.util.Arrays.stream(method.getAnnotations()))
                .filter(annotation -> annotation instanceof org.springframework.web.bind.annotation.PostMapping)
                .map(annotation -> (org.springframework.web.bind.annotation.PostMapping) annotation)
                .flatMap(mapping -> java.util.Arrays.stream(mapping.value()))
                .anyMatch(path -> path.contains("resend-email"));

        assertThat(hasResendMapping).isFalse();
    }
}

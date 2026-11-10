package com.portfolio.photographer.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.portfolio.photographer.dto.BookingResponse;
import com.portfolio.photographer.dto.BookingStatusUpdateRequest;
import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.repository.PackageItemRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.EnumSource;
import org.junit.jupiter.params.provider.MethodSource;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.util.List;
import java.util.Optional;
import java.util.stream.Stream;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class BookingServiceTest {

    private BookingRepository repository;
    private BookingService service;
    private Booking booking;

    @BeforeEach
    void setUp() {
        repository = mock(BookingRepository.class);
        service = new BookingService(
                repository,
                mock(PackageItemRepository.class),
                mock(TermsService.class),
                mock(org.springframework.context.ApplicationEventPublisher.class));

        booking = new Booking();
        booking.setId(11L);
        booking.setName("Raka");
        booking.setEmail("raka@example.com");
        booking.setEventDate(LocalDate.of(2026, 12, 20));
        booking.setEventTime(LocalTime.of(8, 0));
        booking.setStatus(BookingStatus.PENDING);

        when(repository.findById(11L)).thenReturn(Optional.of(booking));
        when(repository.save(any(Booking.class))).thenAnswer(call -> call.getArgument(0));
    }

    static Stream<Arguments> allowedTransitions() {
        return Stream.of(
                Arguments.of(BookingStatus.PENDING, BookingStatus.ACCEPTED),
                Arguments.of(BookingStatus.PENDING, BookingStatus.RESCHEDULE_REQUESTED),
                Arguments.of(BookingStatus.ACCEPTED, BookingStatus.COMPLETED),
                Arguments.of(BookingStatus.ACCEPTED, BookingStatus.RESCHEDULE_REQUESTED),
                Arguments.of(BookingStatus.RESCHEDULE_REQUESTED, BookingStatus.ACCEPTED),
                Arguments.of(BookingStatus.RESCHEDULE_REQUESTED, BookingStatus.COMPLETED)
        );
    }

    static Stream<Arguments> rejectedTransitions() {
        return Stream.of(
                // Backward moves
                Arguments.of(BookingStatus.ACCEPTED, BookingStatus.PENDING),
                Arguments.of(BookingStatus.RESCHEDULE_REQUESTED, BookingStatus.PENDING),
                Arguments.of(BookingStatus.COMPLETED, BookingStatus.PENDING),
                // Terminal status has no way out
                Arguments.of(BookingStatus.COMPLETED, BookingStatus.ACCEPTED),
                Arguments.of(BookingStatus.COMPLETED, BookingStatus.RESCHEDULE_REQUESTED),
                // Skipping the acceptance step
                Arguments.of(BookingStatus.PENDING, BookingStatus.COMPLETED)
        );
    }

    @ParameterizedTest
    @MethodSource("allowedTransitions")
    void allowsLifecycleTransitions(BookingStatus from, BookingStatus to) {
        booking.setStatus(from);

        BookingResponse response = service.updateStatus(
                11L, new BookingStatusUpdateRequest(to));

        assertThat(booking.getStatus()).isEqualTo(to);
        assertThat(response.status()).isEqualTo(to);
        verify(repository).save(booking);
    }

    @ParameterizedTest
    @MethodSource("rejectedTransitions")
    void rejectsInvalidTransitionsWithIndonesianMessage(BookingStatus from, BookingStatus to) {
        booking.setStatus(from);

        assertThatThrownBy(() -> service.updateStatus(
                11L, new BookingStatusUpdateRequest(to)))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("tidak dapat diubah");

        assertThat(booking.getStatus()).isEqualTo(from);
        verify(repository, never()).save(any(Booking.class));
    }

    @ParameterizedTest
    @EnumSource(BookingStatus.class)
    void repeatedTransitionToTheSameStatusIsRejected(BookingStatus from) {
        booking.setStatus(from);

        assertThatThrownBy(() -> service.updateStatus(
                11L, new BookingStatusUpdateRequest(from)))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("sudah berstatus");

        verify(repository, never()).save(any(Booking.class));
    }

    @Test
    void nullStatusIsRejected() {
        assertThatThrownBy(() -> service.validateTransition(BookingStatus.PENDING, null))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("wajib diisi");
    }

    @Test
    void unknownStatusValueCannotBeDeserialized() {
        ObjectMapper mapper = new ObjectMapper();

        assertThatThrownBy(() -> mapper.readValue(
                "{\"status\":\"CANCELLED\"}", BookingStatusUpdateRequest.class))
                .isInstanceOf(com.fasterxml.jackson.databind.exc.InvalidFormatException.class);
    }

    @Test
    void acceptMovesPendingBookingToAcceptedWithoutSendingAnything() {
        BookingResponse response = service.accept(11L);

        assertThat(booking.getStatus()).isEqualTo(BookingStatus.ACCEPTED);
        assertThat(response.status()).isEqualTo(BookingStatus.ACCEPTED);
        assertThat(booking.getConfirmationEmailSentAt()).isNull();
        assertThat(booking.getConfirmationEmailCount()).isZero();
    }

    @Test
    void acceptRejectsCompletedBooking() {
        booking.setStatus(BookingStatus.COMPLETED);

        assertThatThrownBy(() -> service.accept(11L))
                .isInstanceOf(BadRequestException.class)
                .hasMessageContaining("tidak dapat diubah");

        assertThat(booking.getStatus()).isEqualTo(BookingStatus.COMPLETED);
    }

    @Test
    void recordConfirmationEmailSentStoresTimestampAndCount() {
        LocalDateTime before = LocalDateTime.now().minusSeconds(1);

        BookingResponse response = service.recordConfirmationEmailSent(11L);

        assertThat(booking.getConfirmationEmailSentAt()).isAfter(before);
        assertThat(booking.getConfirmationEmailCount()).isEqualTo(1);
        assertThat(response.confirmationEmailCount()).isEqualTo(1);
        assertThat(response.confirmationEmailSentAt()).isNotNull();
    }

    @Test
    void legacyBookingWithoutCountStartsAtZero() {
        booking.setConfirmationEmailCount(null);

        assertThat(booking.getConfirmationEmailCount()).isZero();

        service.recordConfirmationEmailSent(11L);

        assertThat(booking.getConfirmationEmailCount()).isEqualTo(1);
    }

    @Test
    void recordConfirmationEmailForUnknownBookingFails() {
        when(repository.findById(99L)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> service.recordConfirmationEmailSent(99L))
                .isInstanceOf(ResourceNotFoundException.class);
    }

    @Test
    void listResponseExposesTrackingForLegacyRows() {
        booking.setConfirmationEmailCount(null);
        when(repository.findAllByOrderByCreatedAtDesc()).thenReturn(List.of(booking));

        List<BookingResponse> all = service.getAll();

        assertThat(all).hasSize(1);
        assertThat(all.get(0).confirmationEmailCount()).isZero();
        assertThat(all.get(0).confirmationEmailSentAt()).isNull();
    }
}


package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.BookingRequest;
import com.portfolio.photographer.dto.BookingResponse;
import com.portfolio.photographer.dto.BookingStatusUpdateRequest;
import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.event.BookingCreatedEvent;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.entity.PackageItem;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.repository.PackageItemRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.EnumMap;
import java.util.EnumSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

@Service
@RequiredArgsConstructor
public class BookingService {

    /**
     * The booking lifecycle as the project actually models it: a new request is
     * pending, becomes accepted (or asks for a new schedule), and finally ends as
     * completed. COMPLETED is terminal, so nothing moves out of it.
     */
    private static final Map<BookingStatus, Set<BookingStatus>> ALLOWED_TRANSITIONS =
            allowedTransitions();

    private static final Map<BookingStatus, String> STATUS_LABELS = Map.of(
            BookingStatus.PENDING, "Menunggu",
            BookingStatus.ACCEPTED, "Diterima",
            BookingStatus.RESCHEDULE_REQUESTED, "Minta Ubah Jadwal",
            BookingStatus.COMPLETED, "Selesai"
    );

    private final BookingRepository bookingRepository;
    private final PackageItemRepository packageItemRepository;
    private final TermsService termsService;
    private final ApplicationEventPublisher eventPublisher;

    private static Map<BookingStatus, Set<BookingStatus>> allowedTransitions() {
        Map<BookingStatus, Set<BookingStatus>> map = new EnumMap<>(BookingStatus.class);
        map.put(BookingStatus.PENDING, EnumSet.of(
                BookingStatus.ACCEPTED,
                BookingStatus.RESCHEDULE_REQUESTED));
        map.put(BookingStatus.ACCEPTED, EnumSet.of(
                BookingStatus.COMPLETED,
                BookingStatus.RESCHEDULE_REQUESTED));
        map.put(BookingStatus.RESCHEDULE_REQUESTED, EnumSet.of(
                BookingStatus.ACCEPTED,
                BookingStatus.COMPLETED));
        map.put(BookingStatus.COMPLETED, EnumSet.noneOf(BookingStatus.class));
        return map;
    }

    public List<BookingResponse> getAll() {
        return bookingRepository.findAllByOrderByCreatedAtDesc()
                .stream()
                .map(this::toResponse)
                .toList();
    }

    @Transactional
    public BookingResponse create(BookingRequest request) {
        PackageItem selectedPackage = packageItemRepository.findById(request.packageId())
                .orElseThrow(() -> new ResourceNotFoundException(
                        "PackageItem", request.packageId()
                ));

        Booking booking = new Booking();
        booking.setName(request.name());
        booking.setEmail(request.email());
        booking.setPhone(request.phone());
        booking.setEventDate(request.eventDate());
        booking.setEventTime(request.eventTime());
        booking.setPackageTitle(selectedPackage.getTitle());
        booking.setPackagePrice(selectedPackage.getPrice());
        booking.setEventLocation(request.eventLocation());
        booking.setBudget(request.budget());
        booking.setMessage(request.message());
        termsService.validateVersion(request.termsVersion());

        booking.setTermsAccepted(request.termsAccepted());
        booking.setTermsAcceptedAt(java.time.LocalDateTime.now());
        booking.setTermsVersion(request.termsVersion());

        Booking saved = bookingRepository.save(booking);
        eventPublisher.publishEvent(new BookingCreatedEvent(saved));
        return toResponse(saved);
    }

    @Transactional
    public BookingResponse updateStatus(Long id, BookingStatusUpdateRequest request) {
        Booking booking = getBooking(id);
        booking.setStatus(validateTransition(booking.getStatus(), request.status()));
        return toResponse(bookingRepository.save(booking));
    }

    /** Admin accepts a booking. Status only: no e-mail is sent here. */
    @Transactional
    public BookingResponse accept(Long id) {
        Booking booking = getBooking(id);
        booking.setStatus(validateTransition(booking.getStatus(), BookingStatus.ACCEPTED));
        return toResponse(bookingRepository.save(booking));
    }

    /**
     * Persists the result of a successful confirmation e-mail. Callers must only
     * invoke this after SMTP accepted the message, so the timestamp and counter
     * never record a failed delivery.
     */
    @Transactional
    public BookingResponse recordConfirmationEmailSent(Long id) {
        Booking booking = getBooking(id);
        booking.setConfirmationEmailSentAt(LocalDateTime.now());
        booking.setConfirmationEmailCount(booking.getConfirmationEmailCount() + 1);
        return toResponse(bookingRepository.save(booking));
    }

    @Transactional
    public void delete(Long id) {
        if (!bookingRepository.existsById(id)) {
            throw new ResourceNotFoundException("Booking", id);
        }

        bookingRepository.deleteById(id);
    }

    /**
     * Applies the transition allow-list. Returns the target status so callers can
     * assign it, and answers with an Indonesian message when the move is invalid.
     */
    public BookingStatus validateTransition(BookingStatus current, BookingStatus next) {
        if (next == null) {
            throw new BadRequestException("Status pemesanan wajib diisi.");
        }
        if (current == next) {
            throw new BadRequestException(
                    "Pemesanan sudah berstatus \"" + label(current) + "\".");
        }
        Set<BookingStatus> allowed = ALLOWED_TRANSITIONS.getOrDefault(
                current, EnumSet.noneOf(BookingStatus.class));
        if (!allowed.contains(next)) {
            throw new BadRequestException(
                    "Status pemesanan tidak dapat diubah dari \"" + label(current)
                            + "\" menjadi \"" + label(next) + "\".");
        }
        return next;
    }

    private static String label(BookingStatus status) {
        return status == null ? "-" : STATUS_LABELS.getOrDefault(status, status.name());
    }

    private Booking getBooking(Long id) {
        return bookingRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Booking", id));
    }

    private BookingResponse toResponse(Booking b) {
        return new BookingResponse(
                b.getId(),
                b.getName(),
                b.getEmail(),
                b.getPhone(),
                b.getEventDate(),
                b.getEventTime(),
                b.getPackageTitle(),
                b.getPackagePrice(),
                b.getEventLocation(),
                b.getBudget(),
                b.getMessage(),
                b.getStatus(),
                b.getCreatedAt(),
                b.getUpdatedAt(),
                b.getTermsAccepted(),
                b.getTermsAcceptedAt(),
                b.getTermsVersion(),
                b.getConfirmationEmailSentAt(),
                b.getConfirmationEmailCount()
        );
    }
}

package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.BookingResponse;
import com.portfolio.photographer.entity.Booking;
import com.portfolio.photographer.entity.BookingStatus;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.exception.ResourceNotFoundException;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.service.BookingEmailTemplate;
import com.portfolio.photographer.service.BookingService;
import com.portfolio.photographer.service.SmtpMailService;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.*;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Admin booking workflow: schedule changes and client e-mails.
 *
 * <p>Accepting a booking only changes its status. The confirmation e-mail is
 * sent by the admin through a separate, explicit request so a status change can
 * never notify the client by accident.</p>
 */
@RestController
@RequestMapping("/api/admin/bookings")
@RequiredArgsConstructor
public class BookingWorkflowController {

    private final BookingRepository bookingRepository;
    private final BookingService bookingService;
    private final SmtpMailService smtpMailService;
    private final BookingEmailTemplate emailTemplate;

    /** Bookings whose confirmation e-mail is being sent right now. */
    private final Set<Long> sendingConfirmation = ConcurrentHashMap.newKeySet();

    public record RescheduleRequest(@NotNull LocalDate eventDate, @NotNull LocalTime eventTime) {}

    /**
     * Marks a booking as accepted. Deliberately does not send any e-mail: the
     * admin has to press "Kirim Email Konfirmasi" afterwards.
     */
    @PostMapping("/{id}/accept")
    public ResponseEntity<Map<String, Object>> accept(@PathVariable Long id) {
        BookingResponse booking = bookingService.accept(id);
        return ResponseEntity.ok(Map.of(
                "message", "Pemesanan ditandai sebagai diterima. Email konfirmasi belum dikirim.",
                "status", booking.status()));
    }

    /**
     * Sends the acceptance e-mail for an already accepted booking. Only
     * reachable from the authenticated admin panel. The sent timestamp and
     * counter are written after SMTP accepts the message, so a failure leaves
     * both untouched and the admin can retry.
     */
    @PostMapping("/{id}/confirmation-email")
    public ResponseEntity<Map<String, Object>> sendConfirmationEmail(@PathVariable Long id) {
        Booking b = getBooking(id);
        if (b.getStatus() != BookingStatus.ACCEPTED) {
            throw new BadRequestException(
                    "Email konfirmasi hanya dapat dikirim untuk pemesanan yang sudah diterima.");
        }
        if (b.getEmail() == null || b.getEmail().isBlank()) {
            throw new BadRequestException(
                    "Pemesanan ini tidak memiliki alamat email klien, sehingga konfirmasi tidak dapat dikirim.");
        }
        if (!sendingConfirmation.add(id)) {
            throw new BadRequestException(
                    "Email konfirmasi untuk pemesanan ini sedang dikirim. Tunggu hingga proses selesai.");
        }
        try {
            sendConfirmation(b);
        } catch (RuntimeException e) {
            sendingConfirmation.remove(id);
            throw new BadRequestException(
                    "Email konfirmasi gagal dikirim. Periksa konfigurasi pada Pengaturan Email, lalu coba lagi.");
        }
        // Persisted while the in-flight guard still holds, so a second click
        // cannot slip a duplicate send in between SMTP success and this write.
        BookingResponse updated;
        try {
            updated = bookingService.recordConfirmationEmailSent(id);
        } catch (RuntimeException e) {
            throw new BadRequestException(
                    "Email konfirmasi terkirim, tetapi status pengiriman gagal disimpan. Silakan periksa data pemesanan.");
        } finally {
            sendingConfirmation.remove(id);
        }
        return ResponseEntity.ok(Map.of(
                "message", "Email konfirmasi berhasil dikirim ke " + b.getEmail() + ".",
                "sentTo", b.getEmail(),
                "confirmationEmailSentAt", updated.confirmationEmailSentAt(),
                "confirmationEmailCount", updated.confirmationEmailCount()));
    }

    @PatchMapping("/{id}/reschedule")
    @Transactional
    public ResponseEntity<Map<String, Object>> reschedule(@PathVariable Long id, @Valid @RequestBody RescheduleRequest request) {
        Booking b = getBooking(id);
        // A second adjustment while already awaiting a new schedule is a schedule
        // update, not a new transition; everything else must pass the allow-list.
        if (b.getStatus() != BookingStatus.RESCHEDULE_REQUESTED) {
            bookingService.validateTransition(b.getStatus(), BookingStatus.RESCHEDULE_REQUESTED);
        }
        b.setEventDate(request.eventDate());
        b.setEventTime(request.eventTime());
        b.setStatus(BookingStatus.RESCHEDULE_REQUESTED);
        bookingRepository.save(b);
        sendQuietly(b, this::sendReschedule,
                "Jadwal tersimpan, tetapi email pemberitahuan jadwal gagal dikirim. Periksa Pengaturan Email.");
        return ResponseEntity.ok(Map.of(
                "message", "Jadwal diperbarui dan email pemberitahuan jadwal telah dikirim.",
                "status", b.getStatus()));
    }

    private Booking getBooking(Long id) {
        return bookingRepository.findById(id).orElseThrow(() ->
                new ResourceNotFoundException("Pemesanan tidak ditemukan."));
    }

    private void sendQuietly(Booking b, java.util.function.Consumer<Booking> sender, String failureMessage) {
        try {
            sender.accept(b);
        } catch (RuntimeException e) {
            throw new BadRequestException(failureMessage);
        }
    }

    private void sendConfirmation(Booking b) {
        BookingEmailTemplate.EmailContent content = emailTemplate.confirmation(
                b, smtpMailService.getSenderName(), smtpMailService.getSenderEmail());
        send(b, content);
    }

    private void sendReschedule(Booking b) {
        BookingEmailTemplate.EmailContent content = emailTemplate.reschedule(
                b, smtpMailService.getSenderName(), smtpMailService.getSenderEmail());
        send(b, content);
    }

    private void send(Booking b, BookingEmailTemplate.EmailContent content) {
        smtpMailService.sendHtml(b.getEmail(), content.subject(), content.html(), content.text());
    }
}

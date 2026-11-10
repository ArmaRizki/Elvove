
package com.portfolio.photographer.dto;

import com.portfolio.photographer.entity.BookingStatus;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;

/**
 * Full booking view for the admin panel. The package title and price are the
 * values captured when the client booked, so they stay accurate even if the
 * package is later edited or deleted.
 *
 * <p>{@code confirmationEmailSentAt} and {@code confirmationEmailCount} record
 * the last successful confirmation e-mail only; a failed send leaves them
 * unchanged.</p>
 */
public record BookingResponse(
        Long id,
        String name,
        String email,
        String phone,
        LocalDate eventDate,
        LocalTime eventTime,
        String packageTitle,
        BigDecimal packagePrice,
        String eventLocation,
        String budget,
        String message,
        BookingStatus status,
        LocalDateTime createdAt,
        LocalDateTime updatedAt,
        Boolean termsAccepted,
        LocalDateTime termsAcceptedAt,
        String termsVersion,
        LocalDateTime confirmationEmailSentAt,
        int confirmationEmailCount
) {
}

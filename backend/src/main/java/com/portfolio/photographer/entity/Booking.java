
package com.portfolio.photographer.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;

@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "bookings")
public class Booking extends BaseEntity {

    @Column(nullable = false)
    private String name;

    @Column(nullable = false)
    private String email;

    @Column(nullable = false)
    private String phone;

    @Column(nullable = false)
    private LocalDate eventDate;

    private LocalTime eventTime;

    private String packageTitle;

    @Column(precision = 10, scale = 2)
    private BigDecimal packagePrice;

    private String eventLocation;

    private String budget;

    @Column(length = 2000)
    private String message;

    // Terms & Conditions acceptance
    @Column(nullable = false)
    private Boolean termsAccepted = false;

    private LocalDateTime termsAcceptedAt;

    @Column(length = 20)
    private String termsVersion;

    // Confirmation e-mail tracking. Written only after SMTP accepts the message,
    // so a failed delivery never looks like a successful one. Both columns are
    // nullable/absent on rows created before tracking existed.
    private LocalDateTime confirmationEmailSentAt;

    private Integer confirmationEmailCount = 0;

    /** Successful confirmation e-mails sent; 0 for rows created before tracking. */
    public int getConfirmationEmailCount() {
        return confirmationEmailCount == null ? 0 : confirmationEmailCount;
    }

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private BookingStatus status = BookingStatus.PENDING;
}


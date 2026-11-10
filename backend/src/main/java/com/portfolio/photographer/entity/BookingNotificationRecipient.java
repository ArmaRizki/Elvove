package com.portfolio.photographer.entity;

import jakarta.persistence.*;

import java.time.LocalDateTime;

@Entity
@Table(name = "booking_notification_recipients")
public class BookingNotificationRecipient {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false, length = 255) private String email;
    @Column(name = "created_at", nullable = false) private LocalDateTime createdAt;

    public BookingNotificationRecipient() {}
    public BookingNotificationRecipient(String email, LocalDateTime createdAt) {
        this.email = email;
        this.createdAt = createdAt;
    }
    public Long getId() { return id; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public LocalDateTime getCreatedAt() { return createdAt; }
}

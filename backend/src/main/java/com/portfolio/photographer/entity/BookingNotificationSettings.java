package com.portfolio.photographer.entity;

import jakarta.persistence.*;

import java.time.LocalDateTime;

@Entity
@Table(name = "booking_notification_settings")
public class BookingNotificationSettings {
    @Id private Long id;
    @Column(nullable = false) private Boolean enabled;
    @Column(name = "updated_at", nullable = false) private LocalDateTime updatedAt;

    public BookingNotificationSettings() {}
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public Boolean getEnabled() { return enabled; }
    public void setEnabled(Boolean enabled) { this.enabled = enabled; }
    public LocalDateTime getUpdatedAt() { return updatedAt; }
    public void setUpdatedAt(LocalDateTime updatedAt) { this.updatedAt = updatedAt; }
}

package com.portfolio.photographer.event;

import com.portfolio.photographer.entity.Booking;

/**
 * Published when a booking has been written. The listener runs only after the
 * booking transaction commits, so a failed save never sends a notification.
 */
public record BookingCreatedEvent(Booking booking) {}

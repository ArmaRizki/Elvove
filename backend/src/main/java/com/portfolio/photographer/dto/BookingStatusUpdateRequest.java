package com.portfolio.photographer.dto;

import com.portfolio.photographer.entity.BookingStatus;
import jakarta.validation.constraints.NotNull;

public record BookingStatusUpdateRequest(
        @NotNull(message = "Status pemesanan wajib diisi.") BookingStatus status
) {
}

package com.portfolio.photographer.dto;

public record DashboardSummaryResponse(
        long totalPhotos,
        long totalBookings,
        long totalPackages
) {
}

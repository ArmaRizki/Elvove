package com.portfolio.photographer.service;

import com.portfolio.photographer.dto.DashboardSummaryResponse;
import com.portfolio.photographer.repository.BookingRepository;
import com.portfolio.photographer.repository.PackageItemRepository;
import com.portfolio.photographer.repository.PhotoRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class DashboardService {
    private final PhotoRepository photoRepository;
    private final BookingRepository bookingRepository;
    private final PackageItemRepository packageItemRepository;

    public DashboardSummaryResponse getSummary() {
        return new DashboardSummaryResponse(
                photoRepository.count(),
                bookingRepository.count(),
                packageItemRepository.count()
        );
    }
}

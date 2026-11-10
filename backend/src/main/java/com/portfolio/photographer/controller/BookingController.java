package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.BookingRequest;
import com.portfolio.photographer.dto.BookingResponse;
import com.portfolio.photographer.dto.BookingStatusUpdateRequest;
import com.portfolio.photographer.exception.TooManyRequestsException;
import com.portfolio.photographer.security.ClientIpResolver;
import com.portfolio.photographer.security.RateLimiterService;
import com.portfolio.photographer.service.BookingService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class BookingController {

    private final BookingService bookingService;
    private final RateLimiterService rateLimiterService;
    private final ClientIpResolver clientIpResolver;

    private static final int MAX_BOOKINGS_PER_WINDOW = 5;
    private static final long WINDOW_SECONDS = 600; // 10 minutes

    // Public: anyone can submit an inquiry — rate-limited per IP to prevent spam
    @PostMapping("/api/bookings")
    public ResponseEntity<BookingResponse> create(@Valid @RequestBody BookingRequest request, HttpServletRequest httpRequest) {
        String clientIp = clientIpResolver.resolve(httpRequest);
        if (!rateLimiterService.isAllowed("booking:" + clientIp, MAX_BOOKINGS_PER_WINDOW, WINDOW_SECONDS)) {
            throw new TooManyRequestsException("Too many booking requests. Please try again in a few minutes.");
        }
        return ResponseEntity.status(HttpStatus.CREATED).body(bookingService.create(request));
    }

    // Admin-only: viewing/managing inquiries
    @GetMapping("/api/admin/bookings")
    public ResponseEntity<List<BookingResponse>> getAll() {
        return ResponseEntity.ok(bookingService.getAll());
    }

    @PatchMapping("/api/admin/bookings/{id}/status")
    public ResponseEntity<BookingResponse> updateStatus(@PathVariable Long id, @Valid @RequestBody BookingStatusUpdateRequest request) {
        return ResponseEntity.ok(bookingService.updateStatus(id, request));
    }

    @DeleteMapping("/api/admin/bookings/{id}")
    public ResponseEntity<Void> delete(@PathVariable Long id) {
        bookingService.delete(id);
        return ResponseEntity.noContent().build();
    }
}

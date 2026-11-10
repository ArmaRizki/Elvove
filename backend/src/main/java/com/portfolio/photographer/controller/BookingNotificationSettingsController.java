package com.portfolio.photographer.controller;

import com.portfolio.photographer.service.BookingNotificationSettingsService;
import com.portfolio.photographer.service.BookingNotificationSettingsService.Snapshot;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/admin/settings/booking-notifications")
public class BookingNotificationSettingsController {

    private final BookingNotificationSettingsService settingsService;

    public BookingNotificationSettingsController(BookingNotificationSettingsService settingsService) {
        this.settingsService = settingsService;
    }

    /**
     * {@code recipients} is required: an absent field must not silently wipe the
     * saved list. Send an empty array to clear it deliberately.
     */
    public record SaveRequest(
            @NotNull(message = "Status notifikasi wajib diisi.") Boolean enabled,
            @NotNull(message = "Daftar penerima wajib diisi. Kirim array kosong untuk mengosongkannya.")
            List<String> recipients) {}

    @GetMapping
    public ResponseEntity<Snapshot> get() {
        return ResponseEntity.ok(settingsService.current());
    }

    @PutMapping
    public ResponseEntity<Snapshot> save(@Valid @RequestBody SaveRequest request) {
        return ResponseEntity.ok(settingsService.save(request.enabled(), request.recipients()));
    }
}

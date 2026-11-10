package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.PhotoRequest;
import com.portfolio.photographer.dto.PhotoResponse;
import com.portfolio.photographer.service.PhotoService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class PhotoController {

    private final PhotoService photoService;

    @GetMapping("/api/public/photos")
    public ResponseEntity<List<PhotoResponse>> getAll(
            @RequestParam(required = false) Long categoryId,
            @RequestParam(required = false) Boolean featured,
            @RequestParam(required = false) Boolean latest
    ) {
        if (categoryId != null) return ResponseEntity.ok(photoService.getByCategory(categoryId));
        if (Boolean.TRUE.equals(featured)) return ResponseEntity.ok(photoService.getFeatured());
        if (Boolean.TRUE.equals(latest)) return ResponseEntity.ok(photoService.getLatest());
        return ResponseEntity.ok(photoService.getAll());
    }

    @PostMapping("/api/admin/photos")
    public ResponseEntity<PhotoResponse> create(@Valid @RequestBody PhotoRequest request) {
        return ResponseEntity.status(HttpStatus.CREATED).body(photoService.create(request));
    }

    @PutMapping("/api/admin/photos/{id}")
    public ResponseEntity<PhotoResponse> update(@PathVariable Long id, @Valid @RequestBody PhotoRequest request) {
        return ResponseEntity.ok(photoService.update(id, request));
    }

    @DeleteMapping("/api/admin/photos/{id}")
    public ResponseEntity<Void> delete(@PathVariable Long id) {
        photoService.delete(id);
        return ResponseEntity.noContent().build();
    }
}

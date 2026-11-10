package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.SocialLinksRequest;
import com.portfolio.photographer.dto.SocialLinksResponse;
import com.portfolio.photographer.service.SocialLinksService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequiredArgsConstructor
public class SocialLinksController {

    private final SocialLinksService socialLinksService;

    @GetMapping("/api/public/social-links")
    public ResponseEntity<SocialLinksResponse> get() {
        return ResponseEntity.ok(socialLinksService.get());
    }

    @PutMapping("/api/admin/social-links")
    public ResponseEntity<SocialLinksResponse> update(@Valid @RequestBody SocialLinksRequest request) {
        return ResponseEntity.ok(socialLinksService.update(request));
    }
}

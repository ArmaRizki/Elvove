package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.TermsRequest;
import com.portfolio.photographer.dto.TermsResponse;
import com.portfolio.photographer.service.TermsService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequiredArgsConstructor
public class TermsController {

    private final TermsService termsService;

    @GetMapping("/api/public/terms")
    public ResponseEntity<TermsResponse> get() {
        return ResponseEntity.ok(termsService.get());
    }

    @PutMapping("/api/admin/terms")
    public ResponseEntity<TermsResponse> update(
            @Valid @RequestBody TermsRequest request
    ) {
        return ResponseEntity.ok(termsService.update(request));
    }
}
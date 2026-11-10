package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.AboutRequest;
import com.portfolio.photographer.dto.AboutResponse;
import com.portfolio.photographer.service.AboutService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequiredArgsConstructor
public class AboutController {

    private final AboutService aboutService;

    @GetMapping("/api/public/about")
    public ResponseEntity<AboutResponse> get() {
        return ResponseEntity.ok(aboutService.get());
    }

    @PutMapping("/api/admin/about")
    public ResponseEntity<AboutResponse> update(@Valid @RequestBody AboutRequest request) {
        return ResponseEntity.ok(aboutService.update(request));
    }
}

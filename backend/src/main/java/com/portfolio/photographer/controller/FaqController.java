package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.FaqRequest;
import com.portfolio.photographer.dto.FaqResponse;
import com.portfolio.photographer.service.FaqService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class FaqController {

    private final FaqService faqService;

    @GetMapping("/api/public/faqs")
    public ResponseEntity<List<FaqResponse>> getAll() {
        return ResponseEntity.ok(faqService.getAll());
    }

    @PostMapping("/api/admin/faqs")
    public ResponseEntity<FaqResponse> create(@Valid @RequestBody FaqRequest request) {
        return ResponseEntity.status(HttpStatus.CREATED).body(faqService.create(request));
    }

    @PutMapping("/api/admin/faqs/{id}")
    public ResponseEntity<FaqResponse> update(@PathVariable Long id, @Valid @RequestBody FaqRequest request) {
        return ResponseEntity.ok(faqService.update(id, request));
    }

    @DeleteMapping("/api/admin/faqs/{id}")
    public ResponseEntity<Void> delete(@PathVariable Long id) {
        faqService.delete(id);
        return ResponseEntity.noContent().build();
    }
}

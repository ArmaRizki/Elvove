package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.ContactRequest;
import com.portfolio.photographer.dto.ContactResponse;
import com.portfolio.photographer.service.ContactService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequiredArgsConstructor
public class ContactController {

    private final ContactService contactService;

    @GetMapping("/api/public/contact")
    public ResponseEntity<ContactResponse> get() {
        return ResponseEntity.ok(contactService.get());
    }

    @PutMapping("/api/admin/contact")
    public ResponseEntity<ContactResponse> update(@Valid @RequestBody ContactRequest request) {
        return ResponseEntity.ok(contactService.update(request));
    }
}

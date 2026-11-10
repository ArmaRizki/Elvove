package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.PackageItemRequest;
import com.portfolio.photographer.dto.PackageItemResponse;
import com.portfolio.photographer.service.PackageItemService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class PackageItemController {

    private final PackageItemService packageItemService;

    @GetMapping("/api/public/packages")
    public ResponseEntity<List<PackageItemResponse>> getAll() {
        return ResponseEntity.ok(packageItemService.getAll());
    }

    @PostMapping("/api/admin/packages")
    public ResponseEntity<PackageItemResponse> create(@Valid @RequestBody PackageItemRequest request) {
        return ResponseEntity.status(HttpStatus.CREATED).body(packageItemService.create(request));
    }

    @PutMapping("/api/admin/packages/{id}")
    public ResponseEntity<PackageItemResponse> update(@PathVariable Long id, @Valid @RequestBody PackageItemRequest request) {
        return ResponseEntity.ok(packageItemService.update(id, request));
    }

    @DeleteMapping("/api/admin/packages/{id}")
    public ResponseEntity<Void> delete(@PathVariable Long id) {
        packageItemService.delete(id);
        return ResponseEntity.noContent().build();
    }
}

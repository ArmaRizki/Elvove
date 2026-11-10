package com.portfolio.photographer.controller;

import com.portfolio.photographer.dto.FileUploadResponse;
import com.portfolio.photographer.service.FileStorageService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

@RestController
@RequiredArgsConstructor
public class FileUploadController {

    private final FileStorageService fileStorageService;

    @PostMapping("/api/admin/upload")
    public ResponseEntity<FileUploadResponse> upload(@RequestParam("file") MultipartFile file) {
        String url = fileStorageService.store(file);
        return ResponseEntity.ok(new FileUploadResponse(url));
    }
}

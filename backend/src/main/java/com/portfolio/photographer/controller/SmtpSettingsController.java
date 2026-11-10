package com.portfolio.photographer.controller;

import com.portfolio.photographer.entity.SmtpSettings;
import com.portfolio.photographer.repository.SmtpSettingsRepository;
import com.portfolio.photographer.service.SmtpMailService;
import jakarta.validation.Valid;
import jakarta.validation.constraints.*;
import org.springframework.http.ResponseEntity;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.*;
import java.util.Map;

@RestController
@RequestMapping("/api/admin/settings/smtp")
public class SmtpSettingsController {
    private final SmtpSettingsRepository repository;
    private final SmtpMailService mailService;

    public SmtpSettingsController(SmtpSettingsRepository repository, SmtpMailService mailService) {
        this.repository = repository;
        this.mailService = mailService;
    }

    public record SmtpRequest(
        @NotBlank String host,
        @NotNull @Min(1) @Max(65535) Integer port,
        String username,
        String password,
        @NotBlank String encryption,
        @NotNull Boolean authentication,
        @NotBlank String senderName,
        @NotBlank @Email String senderEmail
    ) {}

    @GetMapping
    public ResponseEntity<Map<String, Object>> get() {
        return repository.findById(1L).map(s -> ResponseEntity.ok(response(s)))
            .orElseGet(() -> ResponseEntity.ok(Map.of(
                "configured", false, "host", "smtp.gmail.com", "port", 587,
                "username", "", "passwordConfigured", false, "encryption", "STARTTLS",
                "authentication", true, "senderName", "Elvove",
                "senderEmail", "elvo.vee@gmail.com"
            )));
    }

    @PutMapping
    @Transactional
    public ResponseEntity<Map<String, Object>> save(@Valid @RequestBody SmtpRequest request) {
        String mode = request.encryption().toUpperCase();
        if (!mode.equals("NONE") && !mode.equals("SSL") && !mode.equals("STARTTLS"))
            throw new IllegalArgumentException("Encryption harus NONE, SSL, atau STARTTLS.");

        SmtpSettings s = repository.findById(1L).orElseGet(SmtpSettings::new);
        s.setId(1L);
        s.setHost(request.host().trim());
        s.setPort(request.port());
        s.setUsername(request.username() == null ? "" : request.username().trim());
        s.setEncryption(mode);
        s.setAuthentication(request.authentication());
        s.setSenderName(request.senderName().trim());
        s.setSenderEmail(request.senderEmail().trim());
        if (request.password() != null && !request.password().isBlank()) {
            s.setEncryptedPassword(mailService.encryptPassword(request.password()));
        } else if (s.getEncryptedPassword() == null && Boolean.TRUE.equals(request.authentication())) {
            throw new IllegalArgumentException("Password SMTP wajib diisi untuk autentikasi.");
        }
        repository.save(s);
        return ResponseEntity.ok(response(s));
    }

    private Map<String, Object> response(SmtpSettings s) {
        return Map.of(
            "configured", true, "host", s.getHost(), "port", s.getPort(),
            "username", s.getUsername() == null ? "" : s.getUsername(),
            "passwordConfigured", s.getEncryptedPassword() != null && !s.getEncryptedPassword().isBlank(),
            "encryption", s.getEncryption(), "authentication", s.getAuthentication(),
            "senderName", s.getSenderName(), "senderEmail", s.getSenderEmail()
        );
    }
}

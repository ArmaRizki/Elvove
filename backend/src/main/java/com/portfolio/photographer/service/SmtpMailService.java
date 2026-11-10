package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.SmtpSettings;
import com.portfolio.photographer.repository.SmtpSettingsRepository;
import jakarta.mail.internet.MimeMessage;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.mail.javamail.JavaMailSenderImpl;
import org.springframework.mail.javamail.MimeMessageHelper;
import org.springframework.stereotype.Service;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.Properties;

@Service
public class SmtpMailService {
    private static final Logger log = LoggerFactory.getLogger(SmtpMailService.class);

    private final SmtpSettingsRepository repository;
    private final String encryptionSecret;
    private final SecureRandom random = new SecureRandom();

    public SmtpMailService(SmtpSettingsRepository repository,
            @Value("${app.smtp.encryption-key}") String encryptionSecret) {
        if (encryptionSecret == null || encryptionSecret.isBlank()) {
            throw new IllegalStateException("SMTP_ENCRYPTION_KEY must be set");
        }
        if (encryptionSecret.getBytes(StandardCharsets.UTF_8).length < 32) {
            throw new IllegalStateException("SMTP_ENCRYPTION_KEY must be at least 32 bytes");
        }
        this.repository = repository;
        this.encryptionSecret = encryptionSecret;
    }

    public String getSenderName() { return current().getSenderName(); }

    public String getSenderEmail() { return current().getSenderEmail(); }

    /**
     * Sends a multipart e-mail: the HTML part is what a mail client shows and
     * the plain-text part is the fallback for clients that cannot render HTML.
     */
    public void sendHtml(String to, String subject, String html, String plainText) {
        String host = "unknown";
        String port = "unknown";
        try {
            SmtpSettings settings = current();
            host = settings.getHost();
            port = String.valueOf(settings.getPort());
            JavaMailSenderImpl sender = buildSender(settings);
            MimeMessage message = sender.createMimeMessage();
            // Multipart mode: the helper builds the multipart/alternative
            // message that carries the plain-text and the HTML part.
            MimeMessageHelper helper = new MimeMessageHelper(message, true, StandardCharsets.UTF_8.name());
            helper.setFrom(settings.getSenderEmail(), settings.getSenderName());
            // Replies go back to the same business inbox the message came from.
            helper.setReplyTo(settings.getSenderEmail());
            helper.setTo(to);
            helper.setSubject(subject);
            helper.setText(plainText == null ? "" : plainText, html == null ? "" : html);
            sender.send(message);
            log.info("Email HTML terkirim ke {} melalui {}:{} (subject: {})", to, host, port, subject);
        } catch (Exception e) {
            log.error("Gagal mengirim email HTML ke {} melalui {}:{} - {}", to, host, port, safe(e), e);
            throw new IllegalStateException("Gagal mengirim email SMTP: " + safe(e), e);
        }
    }

    private SmtpSettings current() {
        return repository.findById(1L).orElseThrow(
            () -> new IllegalStateException("SMTP belum dikonfigurasi. Simpan pengaturan SMTP terlebih dahulu."));
    }

    public String encryptPassword(String plain) {
        if (plain == null || plain.isBlank()) return null;
        try {
            byte[] key = MessageDigest.getInstance("SHA-256").digest(encryptionSecret.getBytes(StandardCharsets.UTF_8));
            byte[] iv = new byte[12]; random.nextBytes(iv);
            javax.crypto.Cipher cipher = javax.crypto.Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(javax.crypto.Cipher.ENCRYPT_MODE, new javax.crypto.spec.SecretKeySpec(key, "AES"),
                new javax.crypto.spec.GCMParameterSpec(128, iv));
            byte[] encrypted = cipher.doFinal(plain.getBytes(StandardCharsets.UTF_8));
            ByteBuffer out = ByteBuffer.allocate(iv.length + encrypted.length);
            out.put(iv); out.put(encrypted);
            return Base64.getEncoder().encodeToString(out.array());
        } catch (Exception e) { throw new IllegalStateException("Gagal mengenkripsi password SMTP.", e); }
    }

    private String decryptPassword(String encryptedValue) {
        if (encryptedValue == null || encryptedValue.isBlank()) return "";
        try {
            ByteBuffer in = ByteBuffer.wrap(Base64.getDecoder().decode(encryptedValue));
            byte[] iv = new byte[12]; in.get(iv);
            byte[] ciphertext = new byte[in.remaining()]; in.get(ciphertext);
            byte[] key = MessageDigest.getInstance("SHA-256").digest(encryptionSecret.getBytes(StandardCharsets.UTF_8));
            javax.crypto.Cipher cipher = javax.crypto.Cipher.getInstance("AES/GCM/NoPadding");
            cipher.init(javax.crypto.Cipher.DECRYPT_MODE, new javax.crypto.spec.SecretKeySpec(key, "AES"),
                new javax.crypto.spec.GCMParameterSpec(128, iv));
            return new String(cipher.doFinal(ciphertext), StandardCharsets.UTF_8);
        } catch (Exception e) {
            throw new IllegalStateException("Password SMTP gagal didekripsi. Encryption key mungkin berubah.", e);
        }
    }

    private JavaMailSenderImpl buildSender(SmtpSettings s) {
        JavaMailSenderImpl sender = new JavaMailSenderImpl();
        sender.setHost(s.getHost());
        sender.setPort(s.getPort());
        sender.setUsername(s.getUsername());
        sender.setPassword(Boolean.TRUE.equals(s.getAuthentication()) ? decryptPassword(s.getEncryptedPassword()) : "");
        Properties p = sender.getJavaMailProperties();
        p.put("mail.smtp.auth", String.valueOf(Boolean.TRUE.equals(s.getAuthentication())));
        p.put("mail.smtp.connectiontimeout", "8000");
        p.put("mail.smtp.timeout", "8000");
        p.put("mail.smtp.writetimeout", "8000");
        String mode = s.getEncryption() == null ? "STARTTLS" : s.getEncryption();
        if ("SSL".equalsIgnoreCase(mode)) {
            p.put("mail.smtp.ssl.enable", "true");
            p.put("mail.smtp.starttls.enable", "false");
        } else if ("STARTTLS".equalsIgnoreCase(mode)) {
            p.put("mail.smtp.ssl.enable", "false");
            p.put("mail.smtp.starttls.enable", "true");
            p.put("mail.smtp.starttls.required", "true");
        } else {
            p.put("mail.smtp.ssl.enable", "false");
            p.put("mail.smtp.starttls.enable", "false");
        }
        return sender;
    }

    private String safe(Exception e) {
        return e.getMessage() == null ? e.getClass().getSimpleName() : e.getMessage();
    }
}

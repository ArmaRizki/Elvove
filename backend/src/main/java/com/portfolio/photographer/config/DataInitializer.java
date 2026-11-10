package com.portfolio.photographer.config;

import com.portfolio.photographer.entity.Admin;
import com.portfolio.photographer.repository.AdminRepository;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.CommandLineRunner;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;

import java.util.Locale;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Bootstraps the very first admin account from environment configuration.
 * <p>
 * Runs only when the admins table is empty. It never reads, modifies or saves an
 * existing admin row, so restarts are safe and passwords changed later are kept.
 * Missing or weak configuration fails startup instead of creating a weak account.
 */
@Slf4j
@Component
public class DataInitializer implements CommandLineRunner {

    static final int MIN_PASSWORD_LENGTH = 12;

    // Known default / trivially guessable passwords, compared case-insensitively
    private static final Set<String> DENIED_PASSWORDS = Set.of(
            "admin123", "changeme123!", "password", "password123", "admin", "12345678", "changeme"
    );

    private static final Pattern EMAIL_PATTERN =
            Pattern.compile("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$");

    private final AdminRepository adminRepository;
    private final PasswordEncoder passwordEncoder;
    private final String username;
    private final String email;
    private final String password;

    public DataInitializer(AdminRepository adminRepository,
                           PasswordEncoder passwordEncoder,
                           @Value("${app.admin.username:}") String username,
                           @Value("${app.admin.email:}") String email,
                           @Value("${app.admin.password:}") String password) {
        this.adminRepository = adminRepository;
        this.passwordEncoder = passwordEncoder;
        this.username = username;
        this.email = email;
        this.password = password;
    }

    @Override
    public void run(String... args) {
        if (adminRepository.count() > 0) {
            log.info("Admin account already exists; skipping bootstrap");
            return;
        }

        String trimmedUsername = username == null ? "" : username.trim();
        String trimmedEmail = email == null ? "" : email.trim();

        if (trimmedUsername.isEmpty()) {
            throw new IllegalStateException(
                    "ADMIN_USERNAME must be set to bootstrap the initial admin account");
        }
        if (trimmedEmail.isEmpty()) {
            throw new IllegalStateException(
                    "ADMIN_EMAIL must be set to bootstrap the initial admin account");
        }
        if (!EMAIL_PATTERN.matcher(trimmedEmail).matches()) {
            throw new IllegalStateException("ADMIN_EMAIL is not a valid email address");
        }
        validatePassword(password);

        Admin admin = new Admin();
        admin.setUsername(trimmedUsername);
        admin.setEmail(trimmedEmail);
        admin.setPassword(passwordEncoder.encode(password));
        adminRepository.saveAndFlush(admin);

        log.info("Bootstrapped initial admin account '{}'", trimmedUsername);
    }

    // Messages name the env var only; the value is never included
    private static void validatePassword(String value) {
        if (value == null || value.isBlank()) {
            throw new IllegalStateException(
                    "ADMIN_PASSWORD must be set to bootstrap the initial admin account");
        }
        if (value.length() < MIN_PASSWORD_LENGTH) {
            throw new IllegalStateException(
                    "ADMIN_PASSWORD must be at least " + MIN_PASSWORD_LENGTH + " characters long");
        }
        if (DENIED_PASSWORDS.contains(value.toLowerCase(Locale.ROOT))) {
            throw new IllegalStateException(
                    "ADMIN_PASSWORD is a known default password; choose a strong unique password");
        }
    }
}

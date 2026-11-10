package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.BookingNotificationRecipient;
import com.portfolio.photographer.entity.BookingNotificationSettings;
import com.portfolio.photographer.exception.BadRequestException;
import com.portfolio.photographer.repository.BookingNotificationRecipientRepository;
import com.portfolio.photographer.repository.BookingNotificationSettingsRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.regex.Pattern;

@Service
@RequiredArgsConstructor
public class BookingNotificationSettingsService {

    private static final long SETTINGS_ID = 1L;
    private static final int MAX_RECIPIENTS = 20;
    private static final int MAX_EMAIL_LENGTH = 255;
    private static final Pattern EMAIL = Pattern.compile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$");

    private final BookingNotificationSettingsRepository settingsRepository;
    private final BookingNotificationRecipientRepository recipientRepository;

    /** Enabled flag and recipients, read from the database. */
    @Transactional(readOnly = true)
    public Snapshot current() {
        boolean enabled = settingsRepository.findById(SETTINGS_ID)
                .map(s -> Boolean.TRUE.equals(s.getEnabled()))
                .orElse(false);
        List<String> emails = recipientRepository.findAllByOrderByIdAsc()
                .stream()
                .map(BookingNotificationRecipient::getEmail)
                .toList();
        return new Snapshot(enabled, emails);
    }

    /**
     * Replaces the recipient list and the enabled flag. The list is validated as
     * a whole: a single invalid or duplicate address rejects the entire save, so
     * the stored list never ends up partly updated.
     *
     * <p>The delete must reach the database before the new rows are inserted.
     * Hibernate executes inserts before deletes within a flush, so a plain
     * {@code deleteAll()} leaves the old rows in place while the new ones are
     * inserted; saving a list that keeps an existing address then violates the
     * case-insensitive unique index on {@code lower(email)}. A bulk delete
     * executes immediately, and the explicit flush keeps that ordering even if
     * the repository implementation changes.</p>
     */
    @Transactional
    public Snapshot save(boolean enabled, List<String> rawEmails) {
        List<String> emails = normalize(rawEmails);

        recipientRepository.deleteAllInBatch();
        recipientRepository.flush();

        LocalDateTime now = LocalDateTime.now();
        try {
            recipientRepository.saveAll(
                    emails.stream().map(e -> new BookingNotificationRecipient(e, now)).toList());
            // Surface a constraint violation here rather than at commit time, so
            // it can be reported as invalid input instead of a generic 500.
            recipientRepository.flush();
        } catch (DataIntegrityViolationException e) {
            throw new BadRequestException(
                    "Email penerima duplikat atau tidak valid. Periksa kembali daftar penerima.");
        }

        BookingNotificationSettings settings = settingsRepository.findById(SETTINGS_ID)
                .orElseGet(BookingNotificationSettings::new);
        settings.setId(SETTINGS_ID);
        settings.setEnabled(enabled);
        settings.setUpdatedAt(now);
        settingsRepository.save(settings);

        return new Snapshot(enabled, emails);
    }

    /**
     * Trims each address, rejects blanks, invalid formats and case-insensitive
     * duplicates, and returns the cleaned list in the order it was submitted.
     */
    public List<String> normalize(List<String> rawEmails) {
        if (rawEmails == null) {
            return List.of();
        }
        if (rawEmails.size() > MAX_RECIPIENTS) {
            throw new BadRequestException("Maksimal " + MAX_RECIPIENTS + " alamat email penerima.");
        }

        Map<String, String> seen = new LinkedHashMap<>();
        for (String raw : rawEmails) {
            String email = raw == null ? "" : raw.trim();
            if (email.isEmpty()) {
                throw new BadRequestException("Alamat email penerima tidak boleh kosong.");
            }
            if (email.length() > MAX_EMAIL_LENGTH || !EMAIL.matcher(email).matches()) {
                throw new BadRequestException("Format email tidak valid: " + email);
            }
            String key = email.toLowerCase(Locale.ROOT);
            if (seen.containsKey(key)) {
                throw new BadRequestException("Email penerima duplikat: " + email);
            }
            seen.put(key, email);
        }
        return List.copyOf(seen.values());
    }

    public record Snapshot(boolean enabled, List<String> recipients) {}
}

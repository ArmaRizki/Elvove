package com.portfolio.photographer.service;

import com.portfolio.photographer.entity.Booking;
import lombok.RequiredArgsConstructor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.List;

/**
 * Sends the administrator booking notification.
 * Each recipient is tried on its own, so one failure never stops the others.
 * SMTP credentials and exception details are logged by class name only.
 */
@Service
@RequiredArgsConstructor
public class BookingNotificationMailService {
    private static final Logger log = LoggerFactory.getLogger(BookingNotificationMailService.class);

    private final SmtpMailService smtpMailService;
    private final BookingEmailTemplate template;

    @Value("${app.mail.admin-url}")
    private String adminUrl;

    public void sendBookingNotification(Booking booking, List<String> recipients) {
        String link = BookingLinks.adminBookingUrl(adminUrl, booking.getId());
        BookingEmailTemplate.EmailContent content = template.adminNotification(
                booking, smtpMailService.getSenderName(), link);
        for (String recipient : recipients) {
            try {
                smtpMailService.sendHtml(recipient, content.subject(), content.html(), content.text());
            } catch (Exception e) {
                log.warn("Notifikasi booking #{} gagal dikirim ke satu penerima ({})",
                        booking.getId(), e.getClass().getSimpleName());
            }
        }
    }
}

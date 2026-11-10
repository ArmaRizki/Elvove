package com.portfolio.photographer.service;

/**
 * Builds the admin links used in notification e-mails. The base URL comes from
 * configuration and is checked here: it must be an absolute http(s) URL with no
 * user info, so a credential can never be written into an e-mail.
 */
public final class BookingLinks {

    private BookingLinks() {}

    public static String adminBookingUrl(String baseUrl, Long bookingId) {
        if (bookingId == null || bookingId <= 0) {
            throw new IllegalArgumentException("ID booking tidak valid.");
        }
        return validatedBase(baseUrl) + "/admin/bookings?booking=" + bookingId;
    }

    static String validatedBase(String baseUrl) {
        if (baseUrl == null || baseUrl.isBlank()) {
            throw new IllegalStateException("app.mail.admin-url belum diatur.");
        }
        String trimmed = baseUrl.trim().replaceAll("/+$", "");
        java.net.URI uri;
        try {
            uri = new java.net.URI(trimmed);
        } catch (java.net.URISyntaxException e) {
            throw new IllegalStateException("app.mail.admin-url tidak valid.");
        }
        String scheme = uri.getScheme();
        if (!("https".equalsIgnoreCase(scheme) || "http".equalsIgnoreCase(scheme))
                || uri.getHost() == null
                || uri.getRawUserInfo() != null
                || uri.getRawQuery() != null
                || uri.getRawFragment() != null) {
            throw new IllegalStateException("app.mail.admin-url harus berupa URL http(s) tanpa kredensial.");
        }
        return trimmed;
    }
}

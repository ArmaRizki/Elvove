package com.portfolio.photographer.service;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class BookingLinksTest {

    @Test
    void buildsAdminBookingLinkWithIdQuery() {
        assertThat(BookingLinks.adminBookingUrl("https://elvove.site", 42L))
                .isEqualTo("https://elvove.site/admin/bookings?booking=42");
    }

    @Test
    void trailingSlashIsRemovedFromBase() {
        assertThat(BookingLinks.adminBookingUrl("http://localhost:5173/", 7L))
                .isEqualTo("http://localhost:5173/admin/bookings?booking=7");
    }

    @ParameterizedTest
    @ValueSource(strings = {"ftp://elvove.site", "javascript:alert(1)", "elvove.site", "https://user:pw@elvove.site"})
    void rejectsUnsafeBaseUrls(String base) {
        assertThatThrownBy(() -> BookingLinks.adminBookingUrl(base, 1L))
                .isInstanceOf(IllegalStateException.class);
    }

    @Test
    void rejectsBlankBase() {
        assertThatThrownBy(() -> BookingLinks.adminBookingUrl(" ", 1L))
                .isInstanceOf(IllegalStateException.class);
    }

    @Test
    void rejectsInvalidBookingIds() {
        assertThatThrownBy(() -> BookingLinks.adminBookingUrl("https://elvove.site", null))
                .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> BookingLinks.adminBookingUrl("https://elvove.site", 0L))
                .isInstanceOf(IllegalArgumentException.class);
    }
}

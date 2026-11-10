package com.portfolio.photographer.config;

import com.portfolio.photographer.PhotographerPortfolioApplication;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.TimeZone;

import static org.assertj.core.api.Assertions.assertThat;

class TimeZoneConfigurationTest {

    private static final ZoneId WIB = ZoneId.of("Asia/Jakarta");

    @Test
    void applicationZoneIsWesternIndonesianTime() {
        assertThat(PhotographerPortfolioApplication.APP_TIME_ZONE).isEqualTo("Asia/Jakarta");
        assertThat(WIB.getRules().getOffset(Instant.parse("2026-11-09T00:00:00Z")).getTotalSeconds())
                .isEqualTo(7 * 3600);
    }

    @Test
    void settingTheDefaultZoneMakesLocalTimesWib() {
        TimeZone original = TimeZone.getDefault();
        try {
            TimeZone.setDefault(TimeZone.getTimeZone(PhotographerPortfolioApplication.APP_TIME_ZONE));

            // 05:00 UTC is 12:00 WIB: the wall clock the admin and the e-mail show.
            Instant utc = Instant.parse("2026-11-09T05:00:00Z");
            LocalDateTime local = LocalDateTime.ofInstant(utc, TimeZone.getDefault().toZoneId());

            assertThat(local.format(DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss")))
                    .isEqualTo("2026-11-09T12:00:00");
        } finally {
            TimeZone.setDefault(original);
        }
    }
}

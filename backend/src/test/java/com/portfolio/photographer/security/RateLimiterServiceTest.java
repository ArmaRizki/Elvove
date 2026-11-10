package com.portfolio.photographer.security;

import org.junit.jupiter.api.Test;

import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.ZoneOffset;

import static org.assertj.core.api.Assertions.assertThat;

class RateLimiterServiceTest {

    /** Mutable clock for deterministic window tests. */
    static final class MutableClock extends Clock {
        private Instant now = Instant.parse("2026-01-01T00:00:00Z");
        void advance(Duration d) { now = now.plus(d); }
        @Override public ZoneId getZone() { return ZoneOffset.UTC; }
        @Override public Clock withZone(ZoneId zone) { return this; }
        @Override public Instant instant() { return now; }
    }

    @Test
    void isAllowedLimitUnchanged() {
        RateLimiterService limiter = new RateLimiterService();
        for (int i = 0; i < 5; i++) {
            assertThat(limiter.isAllowed("booking:1.2.3.4", 5, 600)).isTrue();
        }
        assertThat(limiter.isAllowed("booking:1.2.3.4", 5, 600)).isFalse();
        assertThat(limiter.isAllowed("booking:5.6.7.8", 5, 600)).isTrue();
    }

    @Test
    void isAllowedWindowExpires() {
        MutableClock clock = new MutableClock();
        RateLimiterService limiter = new RateLimiterService(clock);
        for (int i = 0; i < 5; i++) limiter.isAllowed("k", 5, 600);
        assertThat(limiter.isAllowed("k", 5, 600)).isFalse();
        clock.advance(Duration.ofSeconds(601));
        assertThat(limiter.isAllowed("k", 5, 600)).isTrue();
    }

    @Test
    void isLimitedDoesNotRecord() {
        RateLimiterService limiter = new RateLimiterService();
        for (int i = 0; i < 10; i++) {
            assertThat(limiter.isLimited("u", 1, 900)).isFalse();
        }
        assertThat(limiter.size()).isZero();
    }

    @Test
    void recordThenIsLimited() {
        RateLimiterService limiter = new RateLimiterService();
        for (int i = 0; i < 4; i++) limiter.record("u", 900);
        assertThat(limiter.isLimited("u", 5, 900)).isFalse();
        limiter.record("u", 900);
        assertThat(limiter.isLimited("u", 5, 900)).isTrue();
    }

    @Test
    void resetClears() {
        RateLimiterService limiter = new RateLimiterService();
        for (int i = 0; i < 5; i++) limiter.record("u", 900);
        limiter.reset("u");
        assertThat(limiter.isLimited("u", 5, 900)).isFalse();
        assertThat(limiter.size()).isZero();
    }

    @Test
    void expiredKeysAreEvictedOnAccess() {
        MutableClock clock = new MutableClock();
        RateLimiterService limiter = new RateLimiterService(clock);
        limiter.record("u", 900);
        assertThat(limiter.size()).isEqualTo(1);
        clock.advance(Duration.ofSeconds(901));
        assertThat(limiter.isLimited("u", 1, 900)).isFalse();
        assertThat(limiter.size()).isZero();
    }

    @Test
    void sweepUsesEachKeysOwnWindow() {
        MutableClock clock = new MutableClock();
        RateLimiterService limiter = new RateLimiterService(clock);
        limiter.isAllowed("short", 5, 60);
        limiter.record("long", 900);
        clock.advance(Duration.ofSeconds(120));

        limiter.sweep();

        assertThat(limiter.size()).isEqualTo(1);
        assertThat(limiter.isLimited("long", 1, 900)).isTrue();
    }

    @Test
    void opportunisticSweepRunsAfterInterval() {
        MutableClock clock = new MutableClock();
        RateLimiterService limiter = new RateLimiterService(clock);
        for (int i = 0; i < 100; i++) limiter.isAllowed("ip-" + i, 5, 30);
        assertThat(limiter.size()).isEqualTo(100);

        clock.advance(Duration.ofSeconds(61));
        limiter.isAllowed("fresh", 5, 30); // triggers the sweep

        assertThat(limiter.size()).isEqualTo(1);
    }
}

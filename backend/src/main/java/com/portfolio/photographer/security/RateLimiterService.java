package com.portfolio.photographer.security;

import org.springframework.stereotype.Component;

import java.time.Clock;
import java.time.Instant;
import java.util.ArrayDeque;
import java.util.Deque;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

// Simple in-memory, sliding-window rate limiter for public endpoints prone to abuse
// (booking form, login). No Redis needed for a single-instance deployment;
// if you later scale to multiple backend instances, move this to a shared store.
//
// Memory is bounded: keys whose window becomes empty are dropped, and an
// opportunistic sweep (at most every SWEEP_INTERVAL, or sooner when the map grows
// past SWEEP_KEY_THRESHOLD) prunes every key using that key's own window.
@Component
public class RateLimiterService {

    static final long SWEEP_INTERVAL_MILLIS = 60_000;
    static final int SWEEP_KEY_THRESHOLD = 10_000;

    private final ConcurrentHashMap<String, Window> requestLog = new ConcurrentHashMap<>();
    private final AtomicLong lastSweepMillis;
    private final Clock clock;

    public RateLimiterService() {
        this(Clock.systemUTC());
    }

    // Visible for tests: allows a controllable clock
    RateLimiterService(Clock clock) {
        this.clock = clock;
        this.lastSweepMillis = new AtomicLong(clock.millis());
    }

    /**
     * Checks the limit and records the request if allowed (behaviour unchanged).
     *
     * @return true if the request is allowed, false if the caller has exceeded the limit
     */
    public boolean isAllowed(String key, int maxRequests, long windowSeconds) {
        Instant now = clock.instant();
        maybeSweep(now);
        boolean[] allowed = new boolean[1];
        requestLog.compute(key, (k, window) -> {
            Window w = window != null ? window : new Window();
            w.windowSeconds = windowSeconds;
            w.prune(now);
            if (w.timestamps.size() >= maxRequests) {
                allowed[0] = false;
            } else {
                w.timestamps.addLast(now);
                allowed[0] = true;
            }
            return w.timestamps.isEmpty() ? null : w;
        });
        return allowed[0];
    }

    /**
     * @return true if the key already has at least {@code maxRequests} events in the
     * window. Does not record a new event.
     */
    public boolean isLimited(String key, int maxRequests, long windowSeconds) {
        Instant now = clock.instant();
        maybeSweep(now);
        boolean[] limited = new boolean[1];
        requestLog.computeIfPresent(key, (k, w) -> {
            w.windowSeconds = windowSeconds;
            w.prune(now);
            limited[0] = w.timestamps.size() >= maxRequests;
            return w.timestamps.isEmpty() ? null : w;
        });
        return limited[0];
    }

    /** Records one event for the key now. */
    public void record(String key, long windowSeconds) {
        Instant now = clock.instant();
        maybeSweep(now);
        requestLog.compute(key, (k, window) -> {
            Window w = window != null ? window : new Window();
            w.windowSeconds = windowSeconds;
            w.prune(now);
            w.timestamps.addLast(now);
            return w;
        });
    }

    /** Removes all recorded events for the key. */
    public void reset(String key) {
        requestLog.remove(key);
    }

    // Visible for tests
    int size() {
        return requestLog.size();
    }

    // Visible for tests: forces a full sweep now
    void sweep() {
        Instant now = clock.instant();
        lastSweepMillis.set(now.toEpochMilli());
        for (String key : requestLog.keySet()) {
            requestLog.computeIfPresent(key, (k, w) -> {
                w.prune(now);
                return w.timestamps.isEmpty() ? null : w;
            });
        }
    }

    private void maybeSweep(Instant now) {
        long nowMillis = now.toEpochMilli();
        long last = lastSweepMillis.get();
        boolean due = nowMillis - last >= SWEEP_INTERVAL_MILLIS || requestLog.size() > SWEEP_KEY_THRESHOLD;
        // Only one thread performs the sweep
        if (due && lastSweepMillis.compareAndSet(last, nowMillis)) {
            sweep();
        }
    }

    // Mutated only inside ConcurrentHashMap compute* callbacks, which are atomic per key
    private static final class Window {
        final Deque<Instant> timestamps = new ArrayDeque<>();
        long windowSeconds;

        void prune(Instant now) {
            Instant windowStart = now.minusSeconds(windowSeconds);
            while (!timestamps.isEmpty() && timestamps.peekFirst().isBefore(windowStart)) {
                timestamps.pollFirst();
            }
        }
    }
}

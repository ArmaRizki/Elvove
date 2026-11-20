package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// A sliding window per key, like Java's RateLimiterService: only the events
// inside this key's own window count, so one endpoint's window can never reset
// another's counter.
type counter struct {
	times []time.Time
	span  time.Duration
}

const (
	sweepInterval = time.Minute
	// sweepKeys is how many keys the sweep leaves alone between sweeps. Past it
	// every request sweeps first, so the map follows the traffic it is counting
	// instead of a clock: one key per token and address that is actually in use.
	sweepKeys = 10_000
)

type RateLimiter struct {
	mu        sync.Mutex
	counters  map[string]*counter
	lastSweep time.Time
}

// NewRateLimiter builds the in-memory limiter.
//
// shortcut: the counters live in this process's memory, so a restart clears
// every login lock and a second backend process would count separately. Fine
// for one process and one admin; move the counters into Postgres when either
// stops being true.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{counters: make(map[string]*counter), lastSweep: time.Now()}
}

func (c *counter) prune(now time.Time) {
	cutoff := now.Add(-c.span)
	i := 0
	for i < len(c.times) && c.times[i].Before(cutoff) {
		i++
	}
	c.times = c.times[i:]
}

// live returns the key's window, dropping the events that have expired. It
// answers nil when the key has no events left at all, so a caller that only
// wants to read can skip the write; entry creates the record instead.
func (l *RateLimiter) live(key string, span time.Duration, now time.Time) *counter {
	c := l.counters[key]
	if c == nil {
		return nil
	}
	c.span = span
	c.prune(now)
	if len(c.times) == 0 {
		return nil
	}
	return c
}

// entry returns the key's window, creating the record when the key is new.
func (l *RateLimiter) entry(key string, span time.Duration, now time.Time) *counter {
	if c := l.live(key, span, now); c != nil {
		return c
	}
	return l.reset(key, span)
}

// reset drops whatever the key held and starts a fresh window. Callers hold the
// lock.
func (l *RateLimiter) reset(key string, span time.Duration) *counter {
	c := &counter{span: span}
	l.counters[key] = c
	return c
}

// maybeSweep drops empty keys so the map stays bounded, at most once a minute
// (sooner when it grows past Java's threshold).
func (l *RateLimiter) maybeSweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepInterval && len(l.counters) <= sweepKeys {
		return
	}
	l.lastSweep = now
	for key, c := range l.counters {
		c.prune(now)
		if len(c.times) == 0 {
			delete(l.counters, key)
		}
	}
}

func (l *RateLimiter) Allow(key string, max int, span time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeSweep(now)
	c := l.live(key, span, now)
	if c == nil {
		// The sweep may just have dropped the key, so the record is made now
		// rather than on every retry.
		c = l.reset(key, span)
	}
	if len(c.times) >= max {
		return false
	}
	c.times = append(c.times, now)
	return true
}

func ClientIP(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if value == "" {
		value = r.RemoteAddr
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}

// IsLimited reports whether key already holds max events in the window without
// recording a new one. Java uses this peek for the per-username login limit so a
// successful login never consumes a slot.
func (l *RateLimiter) IsLimited(key string, max int, span time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeSweep(now)
	if c := l.live(key, span, now); c != nil {
		return len(c.times) >= max
	}
	// A key with no event in the window holds 0, so it is not limited. Creating
	// its record here would only turn a read into a write.
	return 0 >= max
}

// Record adds one event for key (Java's RateLimiterService.record).
func (l *RateLimiter) Record(key string, span time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeSweep(now)
	c := l.live(key, span, now)
	if c == nil {
		c = l.reset(key, span)
	}
	c.times = append(c.times, now)
}

// Reset clears the counter for key (Java's RateLimiterService.reset).
func (l *RateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.counters, key)
}

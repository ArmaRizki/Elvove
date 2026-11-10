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
	sweepKeys     = 10_000
)

type RateLimiter struct {
	mu        sync.Mutex
	counters  map[string]*counter
	lastSweep time.Time
}

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

// entry returns the key's window, dropping the events that have expired.
func (l *RateLimiter) entry(key string, span time.Duration, now time.Time) *counter {
	c := l.counters[key]
	if c == nil {
		c = &counter{}
		l.counters[key] = c
	}
	c.span = span
	c.prune(now)
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
	c := l.entry(key, span, now)
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
	return len(l.entry(key, span, now).times) >= max
}

// Record adds one event for key (Java's RateLimiterService.record).
func (l *RateLimiter) Record(key string, span time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeSweep(now)
	c := l.entry(key, span, now)
	c.times = append(c.times, now)
}

// Reset clears the counter for key (Java's RateLimiterService.reset).
func (l *RateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.counters, key)
}

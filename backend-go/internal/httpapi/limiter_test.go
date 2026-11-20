package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterKeepsOneWindowPerKey(t *testing.T) {
	limiter := NewRateLimiter()
	if !limiter.Allow("login-ip:203.0.113.9", 1, 200*time.Millisecond) {
		t.Fatal("first attempt must be allowed")
	}
	// A booking request with its own, shorter window must not reset the login
	// window: Java prunes each key against that key's own windowSeconds.
	limiter.Allow("booking:203.0.113.9", 5, 10*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	limiter.Allow("booking:203.0.113.9", 5, 10*time.Millisecond)
	if limiter.Allow("login-ip:203.0.113.9", 1, 200*time.Millisecond) {
		t.Fatal("login window was reset by the booking endpoint's shorter window")
	}
}

func TestRateLimiterKeepsItsMapBounded(t *testing.T) {
	// The sweep runs at most once a minute, and sooner only when a request finds
	// the map past sweepKeys, so it is driven by traffic in both ranges. A map
	// already over the limit is swept by the next request; a small map is swept
	// by the first request after the interval.
	sweepDue := func(limiter *RateLimiter) {
		limiter.mu.Lock()
		limiter.lastSweep = time.Now().Add(-2 * sweepInterval)
		limiter.mu.Unlock()
	}
	for _, keys := range []int{50, 12_000} {
		limiter := NewRateLimiter()
		// Every request from a new token or address makes a key. One key per
		// request would grow without end, so the sweep must drop the keys
		// whose window has emptied.
		span := 20 * time.Millisecond
		for i := 0; i < keys; i++ {
			limiter.Allow(fmt.Sprintf("gallery:token%05d:list", i), 5, span)
		}
		if keys <= sweepKeys {
			sweepDue(limiter)
		}
		time.Sleep(3 * span)
		limiter.Allow("gallery:fresh:list", 5, span)
		if keys > sweepKeys {
			// That request swept once, then added its own key.
			sweepDue(limiter)
			limiter.Allow("gallery:fresh:list", 5, span)
		}
		limiter.mu.Lock()
		kept := len(limiter.counters)
		limiter.mu.Unlock()
		if kept > 2 {
			t.Fatalf("%d keys: %d kept after their windows emptied, want at most 2", keys, kept)
		}
	}
}

func TestRateLimiterPeekCreatesNoKey(t *testing.T) {
	limiter := NewRateLimiter()
	// The per-username login check peeks before a password is verified, so a
	// stranger can make it run with any username. A peek must not reserve
	// memory for a key it only read.
	if limiter.IsLimited("login-user:nobody", 5, time.Minute) {
		t.Fatal("an unknown key is not limited")
	}
	limiter.mu.Lock()
	keys := len(limiter.counters)
	limiter.mu.Unlock()
	if keys != 0 {
		t.Fatalf("IsLimited created %d keys, want 0", keys)
	}
	limiter.Record("login-user:someone", time.Minute)
	if limiter.IsLimited("login-user:nobody", 5, time.Minute) {
		t.Fatal("another key's events must not count")
	}
	for i := 0; i < 4; i++ {
		limiter.Record("login-user:someone", time.Minute)
	}
	if !limiter.IsLimited("login-user:someone", 5, time.Minute) {
		t.Fatal("5 recorded events must read as limited")
	}
}

func TestClientIPFallsBackToUnknown(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	request.RemoteAddr = ""
	if got := ClientIP(request); got != "unknown" {
		t.Fatalf("ClientIP = %q, want unknown", got)
	}
	request.RemoteAddr = "203.0.113.4:5678"
	if got := ClientIP(request); got != "203.0.113.4" {
		t.Fatalf("ClientIP = %q, want 203.0.113.4", got)
	}
	request.Header.Set("X-Real-IP", "  ")
	if got := ClientIP(request); got != "203.0.113.4" {
		t.Fatalf("blank X-Real-IP must fall back to RemoteAddr, got %q", got)
	}
}

func TestAdminBootstrapValidation(t *testing.T) {
	if err := adminBootstrapError("admin", "admin@example.com", "strong-password-1"); err != nil {
		t.Fatalf("valid configuration rejected: %v", err)
	}
	cases := []struct{ username, email, password string }{
		{"", "admin@example.com", "strong-password-1"},
		{"admin", "", "strong-password-1"},
		{"admin", "not-an-email", "strong-password-1"},
		{"admin", "admin@example.com", ""},
		{"admin", "admin@example.com", "   "},
		{"admin", "admin@example.com", "short"},
		{"admin", "admin@example.com", "Admin123"},
	}
	for _, c := range cases {
		if err := adminBootstrapError(c.username, c.email, c.password); err == nil {
			t.Errorf("adminBootstrapError(%q, %q, %q) should fail", c.username, c.email, c.password)
		}
	}
}

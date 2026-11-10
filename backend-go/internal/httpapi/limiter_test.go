package httpapi

import (
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

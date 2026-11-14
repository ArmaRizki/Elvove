package httpapi

import (
	"errors"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// The two headers JavaMail always adds. Gmail and spam filters weigh them, so a
// message without them can be delayed or binned.
func TestMessageHeadersIncludeDateAndMessageID(t *testing.T) {
	message := buildMIMEMessage(
		`"Elvove" <elvo.vee@gmail.com>`,
		"armarizki.fz@gmail.com",
		"elvo.vee@gmail.com",
		"Booking Baru — Budi",
		"plain body",
		"<p>html body</p>",
	)
	for _, header := range []string{"Date: ", "Message-ID: <"} {
		if !strings.Contains(message, header) {
			t.Errorf("message missing %q header:\n%s", header, message)
		}
	}
	if strings.Contains(message, "Message-ID: <@") {
		t.Errorf("Message-ID has no domain:\n%s", message)
	}
	// The Date header must be RFC 1123 with a numeric zone, as JavaMail writes.
	line := firstLineWith(message, "Date: ")
	stamp, err := time.Parse(time.RFC1123Z, strings.TrimPrefix(line, "Date: "))
	if err != nil {
		t.Fatalf("Date header %q is not RFC1123Z: %v", line, err)
	}
	// The JVM ran with Asia/Jakarta as its default zone, so JavaMail stamped
	// Date in WIB. Keep that so the header does not look like a UTC-only host.
	if _, offset := stamp.Zone(); offset != 7*60*60 {
		t.Errorf("Date offset = %d, want +7h (WIB)", offset)
	}
}

func TestMessageIDDomain(t *testing.T) {
	for _, tc := range []struct{ sender, want string }{
		{"elvo.vee@gmail.com", "gmail.com"},
		{"  elvo.vee@gmail.com  ", "gmail.com"},
		{"", "elvove.site"},
		{"not-an-address", "elvove.site"},
		{"user@", "elvove.site"},
	} {
		if got := messageIDDomain(tc.sender); got != tc.want {
			t.Errorf("messageIDDomain(%q) = %q, want %q", tc.sender, got, tc.want)
		}
	}
}

// A 4xx/transport failure is transient: the retry loop must try again. A 5xx
// means the server refused the message, so retrying only repeats it.
func TestSendWithRetryRetriesTransientFailure(t *testing.T) {
	restore := retryBackoff
	retryBackoff = time.Millisecond
	defer func() { retryBackoff = restore }()
	attempts := 0
	err := sendWithRetry(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("connection reset by peer")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("sendWithRetry returned %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestSendWithRetryStopsAtPermanentFailure(t *testing.T) {
	attempts := 0
	err := sendWithRetry(func() error {
		attempts++
		return &textproto.Error{Code: 550, Msg: "mailbox unavailable"}
	})
	if err == nil {
		t.Fatal("sendWithRetry returned nil for a permanent failure")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1: a 5xx must not be retried", attempts)
	}
}

func TestSendWithRetryGivesUpAfterThreeTransientFailures(t *testing.T) {
	restore := retryBackoff
	retryBackoff = time.Millisecond
	defer func() { retryBackoff = restore }()
	attempts := 0
	err := sendWithRetry(func() error {
		attempts++
		return &textproto.Error{Code: 421, Msg: "service not available"}
	})
	if err == nil {
		t.Fatal("sendWithRetry returned nil, want the last error")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func firstLineWith(text, prefix string) string {
	for _, line := range strings.Split(text, "\r\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

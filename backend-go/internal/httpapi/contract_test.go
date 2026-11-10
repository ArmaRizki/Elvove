package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v (%s)", err, recorder.Body.String())
	}
	return body
}

func TestWriteErrorMatchesJavaApiErrorShape(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusBadRequest, "Pesan uji.")

	body := decodeError(t, recorder)
	for _, key := range []string{"timestamp", "status", "error", "message", "fieldErrors"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("ApiError is missing %q: %#v", key, body)
		}
	}
	if body["error"] != "Bad Request" || body["status"] != float64(400) || body["message"] != "Pesan uji." {
		t.Fatalf("unexpected error body: %#v", body)
	}
	if body["fieldErrors"] != nil {
		t.Fatalf("non-validation errors must carry null fieldErrors, got %#v", body["fieldErrors"])
	}
}

func TestErrorTimestampIsJakartaWallClock(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusNotFound, "x")

	timestamp, _ := decodeError(t, recorder)["timestamp"].(string)
	if strings.ContainsAny(timestamp, "Z+") {
		t.Fatalf("timestamp must be offset-less like Java LocalDateTime, got %q", timestamp)
	}
	parsed, err := time.Parse("2006-01-02T15:04:05.999999999", timestamp)
	if err != nil {
		t.Fatalf("timestamp %q does not parse as LocalDateTime: %v", timestamp, err)
	}
	if drift := time.Since(time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, jakartaZone)); drift > 2*time.Second || drift < -2*time.Second {
		t.Fatalf("timestamp %q is not current Jakarta time (drift %s)", timestamp, drift)
	}
}

func TestValidationErrorShape(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeValidationError(recorder, map[string]string{"username": "Username is required"})

	body := decodeError(t, recorder)
	if body["error"] != "Validation Failed" || body["message"] != "One or more fields are invalid" {
		t.Fatalf("unexpected validation labels: %#v", body)
	}
	fieldErrors, ok := body["fieldErrors"].(map[string]any)
	if !ok || fieldErrors["username"] != "Username is required" {
		t.Fatalf("fieldErrors missing: %#v", body["fieldErrors"])
	}
}

func TestNotFoundErrorEchoesResourceAndID(t *testing.T) {
	recorder := httptest.NewRecorder()
	notFoundError(recorder, "Package", 42)
	body := decodeError(t, recorder)
	if body["status"] != float64(404) || body["message"] != "Package not found with id: 42" {
		t.Fatalf("unexpected not-found body: %#v", body)
	}
}

func TestLoginValidationRejectsBlankFields(t *testing.T) {
	fieldErrors := loginValidationErrors("   ", "")
	if fieldErrors["username"] != "Username is required" || fieldErrors["password"] != "Password is required" {
		t.Fatalf("unexpected field errors: %#v", fieldErrors)
	}
	if len(loginValidationErrors("admin", "rahasia")) != 0 {
		t.Fatal("valid credentials must not produce field errors")
	}
}

func TestStatusLabelMatchesJavaMessages(t *testing.T) {
	cases := map[string]string{
		"PENDING":              "Menunggu",
		"ACCEPTED":             "Diterima",
		"RESCHEDULE_REQUESTED": "Minta Ubah Jadwal",
		"COMPLETED":            "Selesai",
	}
	for status, label := range cases {
		if got := statusLabel(status); got != label {
			t.Errorf("statusLabel(%q) = %q, want %q", status, got, label)
		}
	}
}

func TestLoginLimiterOnlyCountsFailures(t *testing.T) {
	limiter := NewRateLimiter()
	key := "login-user:203.0.113.7|admin"
	for i := 0; i < 5; i++ {
		if limiter.IsLimited(key, 5, time.Minute) {
			t.Fatalf("key locked after %d failures", i)
		}
		limiter.Record(key, time.Minute)
	}
	if !limiter.IsLimited(key, 5, time.Minute) {
		t.Fatal("key should be locked after 5 failures")
	}
	limiter.Reset(key)
	if limiter.IsLimited(key, 5, time.Minute) {
		t.Fatal("reset must clear the failure counter")
	}
}

func TestRequireAdminResponseIsEmptyLikeJava(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&Auth{}).RequireAdmin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run without an admin identity")
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil))
	if recorder.Code != http.StatusUnauthorized || recorder.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %q; Java's entry point answers an empty 401", recorder.Code, recorder.Body.String())
	}
}

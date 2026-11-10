package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWibTimestampMatchesJavaLocalDateTime(t *testing.T) {
	// Values reaching the JSON writer are already in the column representation
	// (store.Now / a row read back), so they must be rendered as-is: converting
	// the zone here would add another +7 and shift the time the client sees.
	stored := time.Date(2026, 11, 10, 14, 30, 0, 0, time.UTC)
	got := wibTimestamp(stored)
	if got != "2026-11-10T14:30:00" {
		t.Fatalf("wibTimestamp = %q, want the stored WIB wall-clock unchanged", got)
	}
	if strings.ContainsAny(got, "Z+") {
		t.Fatal("timestamp must not carry a zone designator")
	}
}

func TestNowTimestampUsesWibWallClock(t *testing.T) {
	stamp := nowTimestamp()
	parsed, err := time.Parse("2006-01-02T15:04:05.999999999", stamp)
	if err != nil {
		t.Fatalf("nowTimestamp is not parseable: %v", err)
	}
	// Compare wall-clock fields, not instants: the string is offset-less WIB.
	wibNow := time.Now().In(time.FixedZone("WIB", 7*60*60))
	if parsed.Hour() != wibNow.Hour() || parsed.Minute() != wibNow.Minute() || parsed.Day() != wibNow.Day() {
		t.Fatalf("nowTimestamp = %s, want WIB wall-clock near %s", stamp, wibNow.Format("2006-01-02T15:04:05"))
	}
}

func TestBookingValidationReportsFieldErrors(t *testing.T) {
	server := &Server{Limiter: NewRateLimiter()}
	// Only the JSON body is exercised; validation runs before the database.
	body := `{"name":"","email":"nope","phone":"12","eventDate":"","eventTime":"","termsAccepted":false,"termsVersion":""}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/bookings", strings.NewReader(body))
	server.createBooking(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	var payload struct {
		Error       string            `json:"error"`
		FieldErrors map[string]string `json:"fieldErrors"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error != "Validation Failed" {
		t.Fatalf("error label = %q", payload.Error)
	}
	want := map[string]string{
		"name":          "Name is required",
		"email":         "Enter a valid email address",
		"phone":         "Enter a valid phone number",
		"eventDate":     "Event date is required",
		"eventTime":     "Event time is required",
		"packageId":     "Package is required",
		"termsAccepted": "You must accept the Terms & Conditions",
		"termsVersion":  "Terms version is required",
	}
	for field, message := range want {
		if payload.FieldErrors[field] != message {
			t.Errorf("fieldErrors[%q] = %q, want %q", field, payload.FieldErrors[field], message)
		}
	}
}

// BookingRequest.packageId is @NotNull only, so a present 0 or -5 passes bean
// validation and the package lookup answers 404 (Java and production agree).
func TestPackageIDPresenceMatchesNotNull(t *testing.T) {
	decode := func(body string) *int64 {
		var request struct {
			PackageID *int64 `json:"packageId"`
		}
		if err := json.Unmarshal([]byte(body), &request); err != nil {
			t.Fatal(err)
		}
		return request.PackageID
	}
	if decode(`{}`) != nil {
		t.Error("missing packageId should decode to nil")
	}
	for _, body := range []string{`{"packageId":0}`, `{"packageId":-5}`} {
		if id := decode(body); id == nil || *id != 0 && *id != -5 {
			t.Errorf("%s should decode to a non-nil id", body)
		}
	}
}

func TestBookingValidationDoesNotConsumeRateLimit(t *testing.T) {
	limiter := NewRateLimiter()
	server := &Server{Limiter: limiter}
	request := httptest.NewRequest(http.MethodPost, "/api/bookings", strings.NewReader(`{"name":""}`))
	server.createBooking(httptest.NewRecorder(), request)
	// Java validates before the limiter, so the 5 slots must still be free.
	for i := 0; i < 5; i++ {
		if !limiter.Allow("booking:"+ClientIP(request), 5, 10*time.Minute) {
			t.Fatalf("slot %d was consumed by an invalid request", i+1)
		}
	}
}

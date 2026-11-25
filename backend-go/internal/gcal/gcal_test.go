package gcal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const calendarID = "primary"

func token(string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return "test-token", nil }
}

func TestUpsertCreatesAndReturnsTheID(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotAuth = r.URL.Path, r.Method, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"evt-1"}`))
	}))
	defer server.Close()

	client := New(token("t"), server.URL)
	start := time.Date(2026, 11, 25, 5, 30, 0, 0, time.UTC)
	end := start.Add(60 * time.Minute)
	id, err := client.Upsert(context.Background(), calendarID, "", Event{
		Summary: "Chelsea · Emerge", Location: "Balairung UGM",
		Start: start, End: end, Status: "confirmed", BookingID: 12,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if id != "evt-1" {
		t.Fatalf("id = %q", id)
	}
	if gotMethod != http.MethodPost || gotPath != "/calendars/primary/events" {
		t.Fatalf("%s %s, want POST /calendars/primary/events", gotMethod, gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	startBody, _ := body["start"].(map[string]any)
	if startBody["dateTime"] != "2026-11-25T05:30:00" || startBody["timeZone"] != "Asia/Jakarta" {
		t.Fatalf("start = %v", startBody)
	}
	if body["status"] != "confirmed" {
		t.Fatalf("status = %v", body["status"])
	}
	props, _ := body["extendedProperties"].(map[string]any)
	private, _ := props["private"].(map[string]any)
	if private["elvoveBookingId"] != "12" {
		t.Fatalf("extendedProperties = %v", props)
	}
}

func TestUpsertUpdatesTheExistingEvent(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"evt-1"}`))
	}))
	defer server.Close()

	client := New(token("t"), server.URL)
	id, err := client.Upsert(context.Background(), calendarID, "evt-1", Event{
		Summary: "x", Start: time.Now(), End: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if id != "evt-1" {
		t.Fatalf("id = %q", id)
	}
	if len(methods) != 1 || methods[0] != "PATCH /calendars/primary/events/evt-1" {
		t.Fatalf("calls = %v", methods)
	}
}

func TestUpsertRecreatesADeletedEvent(t *testing.T) {
	// The photographer deleted the event by hand; the booking must still land in
	// the calendar instead of silently staying out of it.
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"Not Found"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"evt-2"}`))
	}))
	defer server.Close()

	client := New(token("t"), server.URL)
	id, err := client.Upsert(context.Background(), calendarID, "evt-gone", Event{Summary: "x"})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if id != "evt-2" {
		t.Fatalf("id = %q, want the recreated event", id)
	}
	if strings.Join(methods, ",") != "PATCH,POST" {
		t.Fatalf("methods = %v", methods)
	}
}

func TestUpsertReportsARealFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Request had insufficient authentication scopes."}}`))
	}))
	defer server.Close()

	client := New(token("t"), server.URL)
	if _, err := client.Upsert(context.Background(), calendarID, "", Event{Summary: "x"}); err == nil {
		t.Fatal("a 403 must be reported, not hidden")
	} else if !strings.Contains(err.Error(), "insufficient authentication scopes") {
		t.Fatalf("error hides Google's message: %v", err)
	}
}

func TestDeleteAcceptsAnEventThatIsAlreadyGone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := New(token("t"), server.URL)
	if err := client.Delete(context.Background(), calendarID, "evt-gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

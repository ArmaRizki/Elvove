// Package gcal writes bookings into the photographer's Google Calendar.
//
// It replaces the .ics subscription: a calendar app fetched that address hours
// late and only in one direction, while the Calendar API puts the event in the
// calendar the moment the photographer accepts a booking, moves it when the
// schedule changes and removes it with the booking.
//
// The access token comes from the Drive client, which already trades the stored
// refresh token with Google. Nothing here knows about OAuth: it asks for a
// token and calls the API.
package gcal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// Scope is what the consent screen has to grant on top of Drive. Adding it
	// to the connection the site already has avoids a second Google account.
	Scope = "https://www.googleapis.com/auth/calendar.events"

	defaultAPIBase = "https://www.googleapis.com/calendar/v3"

	// TimeZone is the photographer's zone. Sending the wall clock with its zone
	// name is what keeps 05:30 reading as 05:30; computing an instant by hand
	// invites the seven-hour mistake the feed made.
	TimeZone = "Asia/Jakarta"
)

// Client talks to one calendar. A nil token source means "not connected": the
// callers check Enabled and skip rather than fail.
type Client struct {
	token   func(context.Context) (string, error)
	http    *http.Client
	apiBase string
}

func New(token func(context.Context) (string, error), apiBase string) *Client {
	if trimmed := strings.TrimSpace(apiBase); trimmed != "" {
		apiBase = strings.TrimRight(trimmed, "/")
	} else {
		apiBase = defaultAPIBase
	}
	return &Client{token: token, http: &http.Client{Timeout: 30 * time.Second}, apiBase: apiBase}
}

func (c *Client) Enabled() bool { return c != nil && c.token != nil }

// Event is what a booking looks like in the calendar. Start and End are wall
// clock in TimeZone, matching how the database stores them.
type Event struct {
	Summary     string
	Location    string
	Description string
	Start       time.Time
	End         time.Time
	// Status is "confirmed" or "tentative", the two values Google accepts.
	Status    string
	BookingID int64
}

// Upsert creates the event, or updates the one the booking already has.
//
// An event the photographer deleted by hand in Google Calendar answers 404 on
// update; that is not an error, it is a booking with no event any more, so the
// event is created again and the new id returned.
func (c *Client) Upsert(ctx context.Context, calendarID, eventID string, event Event) (string, error) {
	payload, err := json.Marshal(payload(event))
	if err != nil {
		return "", err
	}
	if eventID != "" {
		status, body, err := c.call(ctx, http.MethodPatch, c.eventURL(calendarID, eventID), payload)
		if err == nil && status >= 200 && status < 300 {
			return eventID, nil
		}
		if err == nil && status != http.StatusNotFound && status != http.StatusGone {
			return "", fmt.Errorf("calendar update: %s", describe(status, body))
		}
		if err != nil {
			return "", err
		}
	}
	return c.create(ctx, calendarID, payload)
}

func (c *Client) create(ctx context.Context, calendarID string, payload []byte) (string, error) {
	status, body, err := c.call(ctx, http.MethodPost, c.eventsURL(calendarID), payload)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("calendar create: %s", describe(status, body))
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return "", fmt.Errorf("calendar create: %w", err)
	}
	if created.ID == "" {
		return "", fmt.Errorf("calendar create: the response carried no id")
	}
	return created.ID, nil
}

// Delete removes the event. An event that is already gone is the outcome we
// wanted, so 404 and 410 are not errors.
func (c *Client) Delete(ctx context.Context, calendarID, eventID string) error {
	status, body, err := c.call(ctx, http.MethodDelete, c.eventURL(calendarID, eventID), nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound || status == http.StatusGone || (status >= 200 && status < 300) {
		return nil
	}
	return fmt.Errorf("calendar delete: %s", describe(status, body))
}

func (c *Client) call(ctx context.Context, method, target string, payload []byte) (int, []byte, error) {
	token, err := c.token(ctx)
	if err != nil {
		return 0, nil, err
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	// Error bodies are small; the cap keeps a misbehaving proxy from filling memory.
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return response.StatusCode, raw, err
}

func (c *Client) eventsURL(calendarID string) string {
	return c.apiBase + "/calendars/" + url.PathEscape(calendarID) + "/events"
}

func (c *Client) eventURL(calendarID, eventID string) string {
	return c.eventsURL(calendarID) + "/" + url.PathEscape(eventID)
}

func payload(event Event) map[string]any {
	status := event.Status
	if status != "tentative" {
		status = "confirmed"
	}
	body := map[string]any{
		"summary": event.Summary,
		"start":   map[string]string{"dateTime": event.Start.Format("2006-01-02T15:04:05"), "timeZone": TimeZone},
		"end":     map[string]string{"dateTime": event.End.Format("2006-01-02T15:04:05"), "timeZone": TimeZone},
		"status":  status,
	}
	if event.Location != "" {
		body["location"] = event.Location
	}
	if event.Description != "" {
		body["description"] = event.Description
	}
	if event.BookingID > 0 {
		// Marks the event as ours, so a stray one can be told apart from the
		// photographer's own appointments.
		body["extendedProperties"] = map[string]any{
			"private": map[string]string{"elvoveBookingId": strconv.FormatInt(event.BookingID, 10)},
		}
	}
	return body
}

// describe turns a Google error body into one readable line. The full body is
// JSON with a message field; falling back to a snippet covers a proxy that
// answers HTML instead.
func describe(status int, body []byte) string {
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", status, apiErr.Error.Message)
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	if snippet == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	return fmt.Sprintf("HTTP %d: %s", status, snippet)
}

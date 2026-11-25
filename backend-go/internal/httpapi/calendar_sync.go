package httpapi

// Google Calendar, written through the API instead of a subscription link.
//
// The event is a copy of the booking, never the source of truth: a Google
// outage must not stop the photographer from accepting a booking, so every call
// here logs its failure and leaves the panel action alone. The panel's sync
// button retries the ones that missed.

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/elvove/photography/backend-go/internal/gcal"
	"github.com/jackc/pgx/v5"
)

// calendarTarget is the account's own calendar. A separate calendar would need
// its id configured, and writing a booking the photographer just accepted to
// the calendar she reads is the point of the feature.
const calendarTarget = "primary"

// calendarFallbackMinutes is used when the package has no session length yet,
// so a package the admin has not filled in still produces a two-hour event
// instead of a zero-length one. It moved here from the old .ics feed.
const calendarFallbackMinutes = 120

// calendarTimeout bounds one Google call so a slow API cannot hold an admin
// request open.
const calendarTimeout = 20 * time.Second

// calendarClient borrows the Drive connection. The refresh token carries both
// the Drive and the Calendar scope, so there is one Google connection, not two.
func (s *Server) calendarClient(ctx context.Context) *gcal.Client {
	driveClient := s.driveClientFor(ctx)
	if !driveClient.Enabled() {
		return gcal.New(nil, "")
	}
	return gcal.New(driveClient.AccessToken, "")
}

// pushBookingToCalendar writes one booking to the calendar, creating the event
// on the first call and moving the existing one afterwards.
func (s *Server) pushBookingToCalendar(ctx context.Context, bookingID int64) error {
	ctx, cancel := context.WithTimeout(ctx, calendarTimeout)
	defer cancel()

	client := s.calendarClient(ctx)
	if !client.Enabled() {
		return errors.New("kalender Google belum terhubung")
	}

	var (
		name, packageTitle, location, clock, status, eventID string
		date                                                 time.Time
		minutes                                              int
	)
	// The package lookup is a scalar subquery because package titles are not
	// unique; a join would return several rows for one booking.
	err := s.Store.Pool.QueryRow(ctx, `
		SELECT name, COALESCE(package_title,''), COALESCE(event_location,''),
		       event_date, COALESCE(to_char(event_time,'HH24:MI'),''), status,
		       COALESCE(google_event_id,''),
		       COALESCE((SELECT p.session_minutes FROM packages p WHERE p.title = bookings.package_title ORDER BY p.id LIMIT 1),0)
		FROM bookings WHERE id=$1`, bookingID).
		Scan(&name, &packageTitle, &location, &date, &clock, &status, &eventID, &minutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if clock == "" {
		// Without an hour there is nothing to place on the calendar yet.
		return nil
	}
	// Only a booking the photographer has accepted belongs in her calendar; a
	// request still waiting for a decision would just clutter it.
	if status != "ACCEPTED" && status != "COMPLETED" {
		return nil
	}

	hour, _ := strconv.Atoi(clock[:2])
	minute, _ := strconv.Atoi(clock[3:5])
	// Wall clock in a neutral container: the zone travels in the payload as
	// Asia/Jakarta, the same way the database stores it.
	start := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC)
	if minutes <= 0 {
		minutes = calendarFallbackMinutes
	}
	end := start.Add(time.Duration(minutes) * time.Minute)

	summary := name
	if packageTitle != "" {
		summary += " · " + packageTitle
	}
	description := ""
	if packageTitle != "" {
		description = "Paket " + packageTitle
	}
	eventStatus := "confirmed"

	id, err := client.Upsert(ctx, calendarTarget, eventID, gcal.Event{
		Summary:     summary,
		Location:    location,
		Description: description,
		Start:       start,
		End:         end,
		Status:      eventStatus,
		BookingID:   bookingID,
	})
	if err != nil {
		return err
	}
	if id != eventID {
		if _, err := s.Store.Exec(ctx, `UPDATE bookings SET google_event_id=$1 WHERE id=$2`, id, bookingID); err != nil {
			return err
		}
	}
	return nil
}

// syncCalendarQuietly is the version the booking handlers call: the row is
// already saved, so a calendar failure is only logged.
func (s *Server) syncCalendarQuietly(ctx context.Context, bookingID int64) {
	if err := s.pushBookingToCalendar(ctx, bookingID); err != nil {
		log.Printf("google calendar: booking %d: %v", bookingID, err)
	}
}

// removeBookingFromCalendar deletes the event of a booking that no longer
// exists. The id is read before the row is deleted, so the call must run after.
func (s *Server) removeBookingFromCalendar(ctx context.Context, eventID string) {
	if eventID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, calendarTimeout)
	defer cancel()
	client := s.calendarClient(ctx)
	if !client.Enabled() {
		return
	}
	if err := client.Delete(ctx, calendarTarget, eventID); err != nil {
		log.Printf("google calendar: delete event %s: %v", eventID, err)
	}
}

// syncCalendar fills the calendar with every booking that has a date and an
// hour. It is the button the panel offers after connecting the account, and the
// retry after a Google outage.
func (s *Server) syncCalendar(w http.ResponseWriter, r *http.Request) {
	if !s.calendarClient(r.Context()).Enabled() {
		writeError(w, 409, "Kalender Google belum terhubung. Hubungkan akun Google di Pengaturan Drive terlebih dahulu.")
		return
	}
	// The ids are read first: the row cursor must not stay open while the loop
	// issues one query per booking on the same pool.
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id FROM bookings WHERE event_date IS NOT NULL AND event_time IS NOT NULL ORDER BY id`)
	if err != nil {
		writeError(w, 500, "Kalender tidak dapat disinkronkan.")
		return
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			writeError(w, 500, "Kalender tidak dapat disinkronkan.")
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Kalender tidak dapat disinkronkan.")
		return
	}

	synced, failed := 0, 0
	for _, id := range ids {
		if err := s.pushBookingToCalendar(r.Context(), id); err != nil {
			log.Printf("google calendar: sync booking %d: %v", id, err)
			failed++
			continue
		}
		synced++
	}
	writeJSON(w, 200, map[string]any{"synced": synced, "failed": failed})
}

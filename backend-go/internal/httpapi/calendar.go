package httpapi

// The calendar feed: one .ics address the photographer adds to Google Calendar,
// Apple Calendar or Outlook once, and every booking shows up there afterwards.
//
// A calendar app fetches a subscription with no headers and no login, so the
// address has to carry its own key: a random token in the path. The token is the
// secret and the panel can rotate it; a wrong or missing token answers 404 just
// like the gallery does, so the address cannot be probed.
//
// The feed is built from `bookings` on every fetch rather than stored, so it
// cannot go stale and there is no second copy of client data lying around. The
// only fields that leave the database are the ones a paper calendar already
// shows — name, package, place, date, time — never e-mail, phone or the client's
// message.

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// calendarFallbackMinutes is used when the package has no session length yet,
// so an untouched package still produces a two-hour event instead of a
// zero-length one.
const calendarFallbackMinutes = 120

type calendarEvent struct {
	ID       int64
	Name     string
	Package  string
	Location string
	Date     time.Time
	// Time is "HH:MM" in the photographer's wall clock (WIB). Empty means the
	// booking has no hour yet, which becomes an all-day event.
	Time    string
	Status  string
	Minutes int
}

// getCalendarFeed serves the subscription. It is public on purpose — a calendar
// app cannot log in — and the token is the whole authentication.
func (s *Server) getCalendarFeed(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	token := strings.TrimSuffix(file, ".ics")
	if token == file || len(token) != 32 {
		writeNotFound(w)
		return
	}
	var stored string
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT token FROM calendar_feed WHERE id=1`).Scan(&stored)
	if err != nil || subtle.ConstantTimeCompare([]byte(stored), []byte(token)) != 1 {
		writeNotFound(w)
		return
	}

	// The package lookup is a scalar subquery, not a join: package titles are not
	// unique, and a join would repeat the event once per matching package — two
	// events with the same UID make the whole feed invalid.
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT b.id, b.name, COALESCE(b.package_title,''), COALESCE(b.event_location,''),
		       b.event_date, COALESCE(to_char(b.event_time,'HH24:MI'),''), b.status,
		       COALESCE((SELECT p.session_minutes FROM packages p WHERE p.title=b.package_title ORDER BY p.id LIMIT 1),0)
		FROM bookings b
		WHERE b.event_date IS NOT NULL
		ORDER BY b.event_date, b.event_time, b.id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Kalender tidak dapat dimuat.")
		return
	}
	defer rows.Close()
	events := make([]calendarEvent, 0)
	for rows.Next() {
		var event calendarEvent
		if err := rows.Scan(&event.ID, &event.Name, &event.Package, &event.Location, &event.Date, &event.Time, &event.Status, &event.Minutes); err != nil {
			writeError(w, http.StatusInternalServerError, "Kalender tidak dapat dimuat.")
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "Kalender tidak dapat dimuat.")
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	// Subscribers must never be handed a cached copy: a rescheduled session has
	// to reach the calendar on the next fetch.
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(buildCalendar(events, time.Now())))
}

// getCalendarFeedURL gives the panel the address to copy, creating the token on
// first use.
func (s *Server) getCalendarFeedURL(w http.ResponseWriter, r *http.Request) {
	token, err := s.calendarToken(r.Context(), false)
	if err != nil {
		writeError(w, 500, "Alamat kalender tidak dapat dibuat.")
		return
	}
	writeJSON(w, 200, map[string]string{"url": s.calendarFeedURL(token)})
}

// rotateCalendarFeed replaces the token, so an address that was shared too
// widely stops working. Every subscribed calendar has to be added again.
func (s *Server) rotateCalendarFeed(w http.ResponseWriter, r *http.Request) {
	token, err := s.calendarToken(r.Context(), true)
	if err != nil {
		writeError(w, 500, "Alamat kalender tidak dapat dibuat.")
		return
	}
	writeJSON(w, 200, map[string]string{"url": s.calendarFeedURL(token)})
}

func (s *Server) calendarFeedURL(token string) string {
	base := strings.TrimRight(s.Config.SiteURL, "/")
	if base == "" {
		base = "https://elvove.site"
	}
	return base + "/api/calendar/" + token + ".ics"
}

func (s *Server) calendarToken(ctx context.Context, rotate bool) (string, error) {
	if rotate {
		token := newSelectionToken()
		_, err := s.Store.Pool.Exec(ctx, `INSERT INTO calendar_feed(id,token,created_at) VALUES(1,$1,NOW())
			ON CONFLICT (id) DO UPDATE SET token=EXCLUDED.token, created_at=NOW()`, token)
		return token, err
	}
	var token string
	err := s.Store.Pool.QueryRow(ctx, `SELECT token FROM calendar_feed WHERE id=1`).Scan(&token)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	token = newSelectionToken()
	// Two admin tabs can ask at the same time; the loser reads the winner's
	// token instead of returning one that was never saved.
	if _, err := s.Store.Pool.Exec(ctx, `INSERT INTO calendar_feed(id,token,created_at) VALUES(1,$1,NOW())
		ON CONFLICT (id) DO NOTHING`, token); err != nil {
		return "", err
	}
	err = s.Store.Pool.QueryRow(ctx, `SELECT token FROM calendar_feed WHERE id=1`).Scan(&token)
	return token, err
}

// buildCalendar renders the feed. Times are stated as Asia/Jakarta rather than
// converted to UTC: the database stores the photographer's wall clock, and
// declaring the zone keeps the hour she typed exactly the hour the calendar
// shows.
func buildCalendar(events []calendarEvent, now time.Time) string {
	var b strings.Builder
	line := func(value string) {
		b.WriteString(foldICSLine(value))
		b.WriteString("\r\n")
	}

	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Elvove//Booking Calendar//ID")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:Elvove")
	line("X-WR-TIMEZONE:Asia/Jakarta")
	// Jakarta has not changed offset since 1964, so a single STANDARD block is
	// correct for every booking the site can hold.
	line("BEGIN:VTIMEZONE")
	line("TZID:Asia/Jakarta")
	line("BEGIN:STANDARD")
	line("DTSTART:19700101T000000")
	line("TZOFFSETFROM:+0700")
	line("TZOFFSETTO:+0700")
	line("TZNAME:WIB")
	line("END:STANDARD")
	line("END:VTIMEZONE")

	stamp := now.UTC().Format("20060102T150405Z")
	for _, event := range events {
		line("BEGIN:VEVENT")
		// The UID follows the booking id, so a reschedule updates the existing
		// entry in the subscriber's calendar instead of adding a second one.
		line("UID:booking-" + strconv.FormatInt(event.ID, 10) + "@elvove.site")
		line("DTSTAMP:" + stamp)

		title := event.Name
		if event.Package != "" {
			title += " · " + event.Package
		}
		line("SUMMARY:" + escapeICSText(title))

		if event.Time == "" {
			line("DTSTART;VALUE=DATE:" + event.Date.Format("20060102"))
			line("DTEND;VALUE=DATE:" + event.Date.AddDate(0, 0, 1).Format("20060102"))
		} else {
			start, end, endDate := calendarSpan(event.Date, event.Time, event.Minutes)
			line("DTSTART;TZID=Asia/Jakarta:" + event.Date.Format("20060102") + "T" + start)
			line("DTEND;TZID=Asia/Jakarta:" + endDate.Format("20060102") + "T" + end)
		}
		if event.Location != "" {
			line("LOCATION:" + escapeICSText(event.Location))
		}
		if event.Package != "" {
			line("DESCRIPTION:" + escapeICSText("Paket "+event.Package))
		}
		line("STATUS:" + calendarStatus(event.Status))
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return b.String()
}

// calendarSpan turns "09:30" plus a length into the start and end clock times,
// moving the end into the next day when a long session crosses midnight.
func calendarSpan(date time.Time, clock string, minutes int) (start, end string, endDate time.Time) {
	if minutes <= 0 {
		minutes = calendarFallbackMinutes
	}
	hour, _ := strconv.Atoi(clock[:2])
	minute, _ := strconv.Atoi(clock[3:5])
	start = clock[:2] + clock[3:5] + "00"

	total := hour*60 + minute + minutes
	endDate = date
	for total >= 24*60 {
		total -= 24 * 60
		endDate = endDate.AddDate(0, 0, 1)
	}
	return start, fmt.Sprintf("%02d%02d00", total/60, total%60), endDate
}

// calendarStatus maps the site's statuses onto what a calendar shows: anything
// the photographer has accepted is confirmed, anything still open is tentative.
func calendarStatus(status string) string {
	switch status {
	case "ACCEPTED", "COMPLETED":
		return "CONFIRMED"
	default:
		return "TENTATIVE"
	}
}

// escapeICSText protects the separators RFC 5545 gives meaning to.
func escapeICSText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		";", `\;`,
		",", `\,`,
		"\r\n", `\n`,
		"\n", `\n`,
		"\r", `\n`,
	)
	return replacer.Replace(value)
}

// foldICSLine breaks a line at 75 octets the way RFC 5545 asks, without cutting
// a UTF-8 character in half: a broken rune makes the whole feed unreadable.
func foldICSLine(line string) string {
	const limit = 75
	if len(line) <= limit {
		return line
	}
	var out strings.Builder
	first := true
	for len(line) > 0 {
		width := limit
		if !first {
			// The space beginning a continuation line counts toward the limit.
			width = limit - 1
		}
		if len(line) <= width {
			out.WriteString(line)
			break
		}
		cut := width
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut--
		}
		if cut == 0 {
			cut = width
		}
		out.WriteString(line[:cut])
		out.WriteString("\r\n ")
		line = line[cut:]
		first = false
	}
	return out.String()
}

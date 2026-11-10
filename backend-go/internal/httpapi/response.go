package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/elvove/photography/backend-go/internal/store"
)

// jakartaZone mirrors Java's TimeZone.setDefault(Asia/Jakarta): every timestamp
// the API emits is WIB wall-clock with no offset.
var jakartaZone = time.FixedZone("Asia/Jakarta", 7*60*60)

// ErrorBody mirrors Java's ApiError. fieldErrors is always present (Jackson
// writes it as null for every non-validation error), so clients can rely on the
// key existing.
type ErrorBody struct {
	Timestamp   string            `json:"timestamp"`
	Status      int               `json:"status"`
	Error       string            `json:"error"`
	Message     string            `json:"message"`
	FieldErrors map[string]string `json:"fieldErrors"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorBody{
		Timestamp: nowTimestamp(),
		Status:    status,
		Error:     http.StatusText(status),
		Message:   message,
	})
}

// writeValidationError mirrors GlobalExceptionHandler.handleValidation: a 400
// whose error label is "Validation Failed" and whose fieldErrors carry the
// per-field deny messages.
func writeValidationError(w http.ResponseWriter, fieldErrors map[string]string) {
	writeJSON(w, http.StatusBadRequest, ErrorBody{
		Timestamp:   nowTimestamp(),
		Status:      http.StatusBadRequest,
		Error:       "Validation Failed",
		Message:     "One or more fields are invalid",
		FieldErrors: fieldErrors,
	})
}

func nowTimestamp() string {
	return wibTimestamp(store.Now())
}

// wibTimestamp renders a time the way Jackson renders LocalDateTime under
// Java's Asia/Jakarta default zone: WIB wall-clock, no offset, no zone suffix.
// The value is already in the column representation (see store.WallClock), so it
// is formatted as-is — converting the zone here would add another +7.
func wibTimestamp(value time.Time) string {
	return value.Format("2006-01-02T15:04:05.999999999")
}

// nowWallClock is the current time prepared for the WIB `timestamp without time
// zone` columns (see store.WallClock).
func nowWallClock() time.Time {
	return store.WallClock(time.Now())
}

// todayWIB is midnight WIB of the current day, the baseline for Java's
// @FutureOrPresent on BookingRequest.eventDate. time.Truncate(24h) would snap to
// midnight UTC instead, which lets yesterday's date through between 00:00 and
// 07:00 WIB.
func todayWIB() time.Time {
	wib := store.WallClock(time.Now())
	return time.Date(wib.Year(), wib.Month(), wib.Day(), 0, 0, 0, 0, time.UTC)
}

// eventDateError mirrors @NotNull @FutureOrPresent on BookingRequest.eventDate
// and returns the fieldErrors message, or "" when the date is acceptable.
func eventDateError(value string) string {
	if value == "" {
		return "Event date is required"
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return "Event date must be a valid date"
	}
	if parsed.Before(todayWIB()) {
		return "Event date must be today or later"
	}
	return ""
}

// notFoundError reproduces Java's ResourceNotFoundException(resourceName, id),
// whose message always ends with the id that was asked for.
func notFoundError(w http.ResponseWriter, resource string, id int64) {
	writeError(w, http.StatusNotFound, resource+" not found with id: "+strconv.FormatInt(id, 10))
}

func writeEmpty(w http.ResponseWriter, status int) { w.WriteHeader(status) }

// writeNotFound mirrors GlobalExceptionHandler.handleNoResource: Java answers an
// unknown path with the same ApiError body as any other 404, not Spring's HTML
// error page, so clients can read `message` on every failure.
func writeNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "Halaman atau sumber daya tidak ditemukan.")
}

// writeUnreadableBody mirrors GlobalExceptionHandler.handleUnreadableBody, which
// covers HttpMessageNotReadableException and MethodArgumentTypeMismatchException:
// a body the server cannot parse is a client error with this exact sentence.
func writeUnreadableBody(w http.ResponseWriter) {
	writeError(w, http.StatusBadRequest, "Format permintaan tidak valid. Periksa kembali data yang dikirim.")
}

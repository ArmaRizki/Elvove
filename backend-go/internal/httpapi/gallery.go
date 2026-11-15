package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elvove/photography/backend-go/internal/drive"
	"github.com/jackc/pgx/v5"
)

// The client photo-selection page.
//
// The raw photos stay in the Drive folder the photographer already uses. This
// page only needs a list of them, a WebP preview of each, and the ids the client
// picked: the previews are uploaded to a "webp" subfolder next to the originals,
// and the picked originals are copied into a "mustedit" folder. Nothing is moved
// or deleted, so a wrong pick never loses a photo.
//
// The page is opened with a per-booking token, and it only exists once the admin
// has pasted the raw folder: until then the link answers "foto belum siap".

// selectionTokenPattern is what a token looks like: 32 random bytes as hex.
var selectionTokenPattern = regexp.MustCompile(`^[a-f0-9]{32,64}$`)

func newSelectionToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

// driveClient is nil when the OAuth credentials are missing; every gallery
// endpoint checks Enabled() and says so instead of failing on a nil pointer.
func (s *Server) driveClient() *drive.Client {
	return s.Drive
}

// galleryBooking is the booking behind one selection link.
type galleryBooking struct {
	ID              int64
	Name            string
	PackageTitle    string
	Limit           int
	RawFolderID     string
	SelectionFolder string
	SubmittedAt     string
}

// loadGalleryBooking reads the booking by token. A malformed token never reaches
// the database: it cannot match, so the query would be wasted work.
func (s *Server) loadGalleryBooking(w http.ResponseWriter, r *http.Request, token string) (galleryBooking, bool) {
	var b galleryBooking
	if !selectionTokenPattern.MatchString(token) {
		writeNotFound(w)
		return b, false
	}
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT b.id, b.name, COALESCE(b.package_title,''),
			COALESCE(b.selection_limit, (SELECT p.photo_selection_limit FROM packages p WHERE p.title = b.package_title ORDER BY p.id LIMIT 1), 0),
			COALESCE(b.raw_folder_id,''), COALESCE(b.selection_folder_id,''),
			COALESCE(to_char(b.selection_submitted_at,'YYYY-MM-DD"T"HH24:MI:SS'),'')
		FROM bookings b WHERE b.selection_token = $1`, token).
		Scan(&b.ID, &b.Name, &b.PackageTitle, &b.Limit, &b.RawFolderID, &b.SelectionFolder, &b.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeNotFound(w)
		return b, false
	}
	if err != nil {
		log.Printf("gallery booking %.8s: %v", token, err)
		writeError(w, 500, "Halaman foto tidak dapat dibuka.")
		return b, false
	}
	return b, true
}

// galleryPhotos refreshes the cached list from Drive, then returns it. New files
// appear, deleted ones disappear.
func (s *Server) galleryPhotos(r *http.Request, booking galleryBooking) ([]drive.File, error) {
	files, err := s.Drive.ListImages(r.Context(), booking.RawFolderID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.ID)
		if _, err := s.Store.Exec(r.Context(), `INSERT INTO booking_photos(booking_id,drive_file_id,file_name,width,height) VALUES($1,$2,$3,$4,$5)
			ON CONFLICT (booking_id,drive_file_id) DO UPDATE SET file_name=EXCLUDED.file_name,width=EXCLUDED.width,height=EXCLUDED.height,updated_at=NOW()`,
			booking.ID, f.ID, f.Name, nullableInt(f.Width), nullableInt(f.Height)); err != nil {
			return nil, err
		}
	}
	// A photo the photographer removed from the folder must also leave the page.
	if _, err := s.Store.Exec(r.Context(), `DELETE FROM booking_photos WHERE booking_id=$1 AND NOT (drive_file_id = ANY($2))`, booking.ID, ids); err != nil {
		return nil, err
	}
	return files, nil
}

// getGallery answers what the client page needs: the photos, the entitlement and
// what is already picked. Previews are converted lazily by galleryPhoto, so this
// call stays fast even for a folder of two hundred photos.
func (s *Server) getGallery(w http.ResponseWriter, r *http.Request) {
	booking, ok := s.loadGalleryBooking(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	if booking.RawFolderID == "" {
		writeError(w, 409, "Foto untuk pemesanan ini belum siap. Silakan hubungi Elvove.")
		return
	}
	if !s.driveClient().Enabled() {
		writeError(w, 503, "Halaman foto sedang tidak tersedia. Silakan hubungi Elvove.")
		return
	}
	files, err := s.galleryPhotos(r, booking)
	if err != nil {
		log.Printf("gallery list booking %d: %v", booking.ID, err)
		writeError(w, 502, "Folder foto tidak dapat dibaca. Silakan hubungi Elvove.")
		return
	}
	selected := map[string]int{}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT drive_file_id, position FROM booking_photos WHERE booking_id=$1 AND selected`, booking.ID)
	if err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat dibaca.")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var position int
		if err := rows.Scan(&id, &position); err == nil {
			selected[id] = position
		}
	}

	type photo struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Selected bool   `json:"selected"`
		Position *int   `json:"position"`
	}
	photos := make([]photo, 0, len(files))
	for _, f := range files {
		item := photo{ID: f.ID, Name: f.Name}
		if position, picked := selected[f.ID]; picked {
			item.Selected = true
			item.Position = &position
		}
		photos = append(photos, item)
	}
	writeJSON(w, 200, map[string]any{
		"clientName":   booking.Name,
		"packageTitle": booking.PackageTitle,
		"limit":        booking.Limit,
		"photos":       photos,
		"submittedAt":  nullableString(booking.SubmittedAt),
	})
}

// galleryPhoto streams one WebP preview, converting it on first request.
func (s *Server) galleryPhoto(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	booking, ok := s.loadGalleryBooking(w, r, token)
	if !ok {
		return
	}
	if booking.RawFolderID == "" || !s.driveClient().Enabled() {
		writeNotFound(w)
		return
	}
	fileID := r.PathValue("id")
	var name, webpID string
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT file_name, COALESCE(webp_file_id,'') FROM booking_photos WHERE booking_id=$1 AND drive_file_id=$2`, booking.ID, fileID).
		Scan(&name, &webpID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeNotFound(w)
		return
	}
	if err != nil {
		writeError(w, 500, "Foto tidak dapat dibuka.")
		return
	}
	// Previews live under UPLOAD_DIR, which is the persistent volume, so they
	// survive a redeploy and Drive is read once per photo. The folder is not
	// pruned automatically: delete gallery/<token> once a booking's edit is
	// delivered if the volume gets tight.
	cache := filepath.Join(s.Config.UploadDir, "gallery", token, fileID+".webp")
	data, err := os.ReadFile(cache)
	if err != nil {
		// One conversion per photo even when the grid fires six requests at once.
		unlock := s.galleryLock(token + "/" + fileID)
		data, webpID, err = s.convertGalleryPhoto(r.Context(), booking, fileID, name, webpID, cache)
		unlock()
		if err != nil {
			log.Printf("gallery convert %d/%s: %v", booking.ID, fileID, err)
			writeError(w, 502, "Foto tidak dapat dimuat.")
			return
		}
	}
	w.Header().Set("Content-Type", "image/webp")
	// The bytes never change for a given photo, so the browser keeps them.
	w.Header().Set("Cache-Control", "private, max-age=604800")
	w.Write(data)
}

// convertGalleryPhoto downloads the original, scales it into a WebP, stores it in
// the Drive "webp" subfolder and caches it on disk. Drive is asked only once per
// photo: after that the local copy answers.
func (s *Server) convertGalleryPhoto(ctx context.Context, booking galleryBooking, fileID, name, webpID, cache string) ([]byte, string, error) {
	body, err := s.Drive.Download(ctx, fileID)
	if err != nil {
		return nil, webpID, err
	}
	defer body.Close()
	dir := filepath.Dir(cache)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, webpID, err
	}
	original := strings.TrimSuffix(cache, ".webp") + ".source"
	out, err := os.OpenFile(original, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return nil, webpID, err
	}
	_, copyErr := io.Copy(out, body)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(original)
		return nil, webpID, copyErr
	}
	if closeErr != nil {
		os.Remove(original)
		return nil, webpID, closeErr
	}
	defer os.Remove(original)

	if err := convertWebP(original, cache, 1600, 78); err != nil {
		return nil, webpID, err
	}
	data, err := os.ReadFile(cache)
	if err != nil {
		return nil, webpID, err
	}
	if webpID == "" {
		folder, folderErr := s.Drive.EnsureFolder(ctx, booking.RawFolderID, "webp")
		if folderErr != nil {
			// The preview still works; only the Drive copy is missing, so the next
			// request retries instead of failing the page.
			log.Printf("gallery webp folder booking %d: %v", booking.ID, folderErr)
			return data, webpID, nil
		}
		uploaded, uploadErr := s.Drive.Upload(ctx, folder, strings.TrimSuffix(name, filepath.Ext(name))+".webp", "image/webp", data)
		if uploadErr != nil {
			log.Printf("gallery webp upload booking %d: %v", booking.ID, uploadErr)
			return data, webpID, nil
		}
		webpID = uploaded
		if _, err := s.Store.Exec(ctx, `UPDATE booking_photos SET webp_file_id=$1,updated_at=NOW() WHERE booking_id=$2 AND drive_file_id=$3`, webpID, booking.ID, fileID); err != nil {
			log.Printf("gallery webp save booking %d: %v", booking.ID, err)
		}
	}
	return data, webpID, nil
}

// saveSelection replaces the picked set. The rules are the ones the page shows:
// no more than the package allows, and only photos that are actually in the
// folder — a stale tab cannot smuggle in an id.
func (s *Server) saveSelection(w http.ResponseWriter, r *http.Request) {
	booking, ok := s.loadGalleryBooking(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	if booking.SubmittedAt != "" {
		writeError(w, 409, "Pilihan foto sudah dikirim dan tidak dapat diubah lagi.")
		return
	}
	var req struct {
		FileIDs []string `json:"fileIds"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	limit := booking.Limit
	if limit < 1 {
		writeError(w, 409, "Paket pemesanan ini tidak mencakup pemilihan foto.")
		return
	}
	seen := make(map[string]bool, len(req.FileIDs))
	for _, id := range req.FileIDs {
		if !selectionFileIDPattern.MatchString(id) || seen[id] {
			writeValidationError(w, map[string]string{"fileIds": "Daftar foto tidak valid."})
			return
		}
		seen[id] = true
	}
	if len(req.FileIDs) > limit {
		writeValidationError(w, map[string]string{"fileIds": fmt.Sprintf("Maksimal %d foto untuk paket ini.", limit)})
		return
	}
	if len(req.FileIDs) > 0 {
		var known int
		if err := s.Store.Pool.QueryRow(r.Context(), `SELECT count(*) FROM booking_photos WHERE booking_id=$1 AND drive_file_id = ANY($2)`, booking.ID, req.FileIDs).Scan(&known); err != nil {
			writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
			return
		}
		if known != len(req.FileIDs) {
			writeValidationError(w, map[string]string{"fileIds": "Ada foto yang tidak lagi tersedia. Muat ulang halaman ini."})
			return
		}
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `UPDATE booking_photos SET selected=FALSE,position=NULL,updated_at=NOW() WHERE booking_id=$1`, booking.ID); err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
		return
	}
	for i, id := range req.FileIDs {
		if _, err := tx.Exec(r.Context(), `UPDATE booking_photos SET selected=TRUE,position=$1,updated_at=NOW() WHERE booking_id=$2 AND drive_file_id=$3`, i+1, booking.ID, id); err != nil {
			writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
		return
	}
	writeJSON(w, 200, map[string]any{"selectedCount": len(req.FileIDs), "limit": limit})
}

var selectionFileIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{10,100}$`)

// submitSelection closes the choice: it creates the "mustedit" folder beside the
// originals and copies the picked full-resolution files into it. Copying rather
// than moving means the raw folder stays intact whatever happens.
func (s *Server) submitSelection(w http.ResponseWriter, r *http.Request) {
	booking, ok := s.loadGalleryBooking(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	if booking.SubmittedAt != "" {
		writeError(w, 409, "Pilihan foto sudah dikirim.")
		return
	}
	if !s.driveClient().Enabled() {
		writeError(w, 503, "Halaman foto sedang tidak tersedia. Silakan hubungi Elvove.")
		return
	}
	if booking.Limit < 1 {
		writeError(w, 409, "Paket pemesanan ini tidak mencakup pemilihan foto.")
		return
	}
	// Two taps on "kirim" (or a retried request) must not copy the files twice
	// in parallel; the second one waits here and then sees the lock below.
	unlock := s.galleryLock(fmt.Sprintf("submit/%d", booking.ID))
	defer unlock()
	var submitted string
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT COALESCE(to_char(selection_submitted_at,'YYYY-MM-DD'),'') FROM bookings WHERE id=$1`, booking.ID).Scan(&submitted); err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat dibaca.")
		return
	}
	if submitted != "" {
		writeError(w, 409, "Pilihan foto sudah dikirim.")
		return
	}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT drive_file_id, file_name FROM booking_photos WHERE booking_id=$1 AND selected ORDER BY position`, booking.ID)
	if err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat dibaca.")
		return
	}
	type picked struct{ ID, Name string }
	items := make([]picked, 0, booking.Limit)
	for rows.Next() {
		var item picked
		if err := rows.Scan(&item.ID, &item.Name); err == nil {
			items = append(items, item)
		}
	}
	rows.Close()
	if len(items) == 0 {
		writeValidationError(w, map[string]string{"fileIds": "Pilih minimal satu foto sebelum mengirim."})
		return
	}
	if len(items) > booking.Limit {
		writeError(w, 409, fmt.Sprintf("Pilihan melebihi %d foto untuk paket ini.", booking.Limit))
		return
	}

	folder, err := s.Drive.EnsureFolder(r.Context(), booking.RawFolderID, "mustedit")
	if err != nil {
		log.Printf("gallery mustedit folder booking %d: %v", booking.ID, err)
		writeError(w, 502, "Folder hasil edit tidak dapat dibuat. Silakan hubungi Elvove.")
		return
	}
	// A retry after a partial failure must not copy the same photo twice, so the
	// folder is read first and names already there are skipped.
	existing, err := s.Drive.ListImages(r.Context(), folder)
	if err != nil {
		log.Printf("gallery mustedit list booking %d: %v", booking.ID, err)
		writeError(w, 502, "Folder hasil edit tidak dapat dibaca. Silakan hubungi Elvove.")
		return
	}
	// Names are counted, not just remembered: Drive allows two files with the
	// same name in one folder, and a count keeps a retry from skipping the
	// second copy of a name the client picked twice.
	already := make(map[string]int, len(existing))
	for _, f := range existing {
		already[f.Name]++
	}
	for _, item := range items {
		if already[item.Name] > 0 {
			already[item.Name]--
			continue
		}
		if _, err := s.Drive.Copy(r.Context(), item.ID, folder, item.Name); err != nil {
			log.Printf("gallery copy %d/%s: %v", booking.ID, item.ID, err)
			writeError(w, 502, "Sebagian foto gagal disalin. Silakan coba kirim lagi.")
			return
		}
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE bookings SET selection_folder_id=$1,selection_submitted_at=NOW(),updated_at=NOW() WHERE id=$2`, folder, booking.ID); err != nil {
		log.Printf("gallery submit booking %d: %v", booking.ID, err)
		writeError(w, 500, "Pilihan foto tersimpan, tetapi status gagal diperbarui. Silakan hubungi Elvove.")
		return
	}
	writeJSON(w, 200, map[string]any{"selectedCount": len(items), "submittedAt": wibTimestamp(nowWallClock()), "message": "Pilihan foto sudah kami terima. Terima kasih."})
}

// updateBookingRawFolder stores the Drive folder holding the raw photos and makes
// sure the booking has a selection link. The folder is checked against Drive
// before it is saved, so a typo is caught here rather than by the client.
func (s *Server) updateBookingRawFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := bookingIDFromPath(w, r)
	if !ok {
		return
	}
	var req struct {
		RawFolderURL *string `json:"rawFolderUrl"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.RawFolderURL == nil {
		writeValidationError(w, map[string]string{"rawFolderUrl": "Link folder foto wajib diisi."})
		return
	}
	link := strings.TrimSpace(*req.RawFolderURL)
	folderID := ""
	if link != "" {
		parsed, err := drive.FolderID(link)
		if err != nil {
			writeValidationError(w, map[string]string{"rawFolderUrl": "Tempel link folder Google Drive atau ID foldernya."})
			return
		}
		folderID = parsed
		if !s.driveClient().Enabled() {
			writeError(w, 503, "Google Drive belum terhubung di server ini.")
			return
		}
		// Reading once proves the folder exists and is shared with this account.
		if _, err := s.Drive.ListImages(r.Context(), folderID); err != nil {
			log.Printf("raw folder check %s: %v", folderID, err)
			writeError(w, 400, "Folder tidak dapat dibuka. Pastikan link benar dan folder dibagikan ke aplikasi Elvove.")
			return
		}
	}
	token := newSelectionToken()
	_, err := s.Store.Exec(r.Context(), `UPDATE bookings SET raw_folder_id=NULLIF($2,''),
			selection_token=COALESCE(selection_token,$3),
			selection_limit=COALESCE(selection_limit,(SELECT photo_selection_limit FROM packages p WHERE p.title=bookings.package_title ORDER BY p.id LIMIT 1),0),
			selection_submitted_at=NULL,selection_folder_id=NULL,updated_at=NOW() WHERE id=$1`, id, folderID, token)
	if err != nil {
		writeError(w, 500, "Booking could not be updated.")
		return
	}
	booking, err := s.fetchBooking(r, id)
	if err != nil {
		writeError(w, 404, "Pemesanan tidak ditemukan.")
		return
	}
	writeJSON(w, 200, booking)
}

// galleryLock serializes the conversion of one photo.
func (s *Server) galleryLock(key string) func() {
	value, _ := s.galleryMu.LoadOrStore(key, &keyedMutex{})
	lock := value.(*keyedMutex)
	lock.Lock()
	return lock.Unlock
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

// keyedMutex is one lock per photo conversion, so two requests for the same
// thumbnail wait instead of both downloading and converting the original.
type keyedMutex struct {
	mu sync.Mutex
}

func (k *keyedMutex) Lock()   { k.mu.Lock() }
func (k *keyedMutex) Unlock() { k.mu.Unlock() }

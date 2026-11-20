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
// and the picked originals are copied into the "foto terpilih" folder. Nothing is moved
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

// galleryBooking is the booking behind one selection link.
type galleryBooking struct {
	ID           int64
	Name         string
	PackageTitle string
	Limit        int
	RawFolderID  string
	WebpFolderID string
	MusteditID   string
	SubmittedAt  string
}

// selectionLimitSQL is the one rule for the photo quota: the snapshot taken
// when the booking was made, and the package's quota when that snapshot is
// empty. The client page, the e-mail and the admin panel all use it, so they
// cannot disagree about how many photos the client may pick.
//
// The alias is the bookings table as it is written in the query.
func selectionLimitSQL(alias string) string {
	return `COALESCE(NULLIF(` + alias + `.selection_limit,0), (SELECT p.photo_selection_limit FROM packages p WHERE p.title = ` + alias + `.package_title ORDER BY p.id LIMIT 1), 0)`
}

// loadGalleryBooking reads the booking by token. A malformed token never reaches
// the database: it cannot match, so the query would be wasted work.
//
// A booking made before packages carried a quota holds 0 in its snapshot, which
// would close its page for good. NULLIF lets such a booking fall back to the
// package's quota; a package that really offers no selection still answers 0.
func (s *Server) loadGalleryBooking(w http.ResponseWriter, r *http.Request, token string) (galleryBooking, bool) {
	var b galleryBooking
	if !selectionTokenPattern.MatchString(token) {
		writeNotFound(w)
		return b, false
	}
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT b.id, b.name, COALESCE(b.package_title,''),
			`+selectionLimitSQL("b")+`,
			COALESCE(b.raw_folder_id,''),
			COALESCE(b.folder_webp_id,''), COALESCE(b.folder_mustedit_id,''),
			COALESCE(to_char(b.selection_submitted_at,'YYYY-MM-DD"T"HH24:MI:SS'),'')
		FROM bookings b WHERE b.selection_token = $1`, token).
		Scan(&b.ID, &b.Name, &b.PackageTitle, &b.Limit, &b.RawFolderID, &b.WebpFolderID, &b.MusteditID, &b.SubmittedAt)
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
func (s *Server) galleryPhotos(r *http.Request, client *drive.Client, booking galleryBooking) ([]drive.File, error) {
	// A reload, or the same page opened twice, reuses the listing for a minute
	// instead of walking the folder tree again.
	s.galleryListMu.Lock()
	for id, entry := range s.galleryListed {
		if time.Since(entry.at) >= listingTTL {
			delete(s.galleryListed, id)
		}
	}
	cached, ok := s.galleryListed[booking.RawFolderID]
	s.galleryListMu.Unlock()
	if ok {
		// The rows are per booking, so they are synced even on a cache hit: a
		// second booking on the same folder would otherwise never get them.
		if err := s.syncBookingPhotos(r.Context(), booking.ID, cached.files); err != nil {
			return nil, err
		}
		return cached.files, nil
	}
	// The photographer sorts the photos into subfolders, so the whole tree is
	// read. The three folders the site keeps for itself are left out.
	files, err := client.ListImagesRecursive(r.Context(), booking.RawFolderID, webpFolderName, pickedFolderName, editedFolderName)
	if err != nil {
		return nil, err
	}
	if err := s.syncBookingPhotos(r.Context(), booking.ID, files); err != nil {
		return nil, err
	}
	s.galleryListMu.Lock()
	s.galleryListed[booking.RawFolderID] = galleryListing{files: files, at: time.Now()}
	s.galleryListMu.Unlock()
	return files, nil
}

// syncBookingPhotos records one folder listing for one booking: files that are
// new are added, ones that left the folder are dropped unless the client picked
// them. The Drive listing itself may be shared with another booking; these rows
// may not.
func (s *Server) syncBookingPhotos(ctx context.Context, bookingID int64, files []drive.File) error {
	ids := make([]string, 0, len(files))
	names := make([]string, 0, len(files))
	widths := make([]int32, 0, len(files))
	heights := make([]int32, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.ID)
		names = append(names, f.Name)
		widths = append(widths, int32(f.Width))
		heights = append(heights, int32(f.Height))
	}
	if len(ids) > 0 {
		// One query for the whole folder, not one per photo.
		if _, err := s.Store.Exec(ctx, `INSERT INTO booking_photos(booking_id,drive_file_id,file_name,width,height)
			SELECT $1, u.id, u.name, NULLIF(u.width,0), NULLIF(u.height,0)
			FROM unnest($2::text[], $3::text[], $4::int[], $5::int[]) AS u(id,name,width,height)
			ON CONFLICT (booking_id,drive_file_id) DO UPDATE SET file_name=EXCLUDED.file_name,width=EXCLUDED.width,height=EXCLUDED.height,updated_at=NOW()`,
			bookingID, ids, names, widths, heights); err != nil {
			return err
		}
	}
	// A photo the photographer removed from the folder leaves the page, but a
	// photo the client already picked stays: that choice is not the
	// photographer's to erase.
	_, err := s.Store.Exec(ctx, `DELETE FROM booking_photos WHERE booking_id=$1 AND NOT selected AND NOT (drive_file_id = ANY($2))`, bookingID, ids)
	return err
}

// galleryListing is one raw folder's Drive listing, kept for listingTTL.
type galleryListing struct {
	files []drive.File
	at    time.Time
}

// listingTTL is how long a listing is reused. A photo added to the folder shows
// up on the client page within this window.
const listingTTL = time.Minute

// galleryConverters is how many originals are downloaded and converted at once.
// Two keep a page moving without exhausting memory on a small server.
const galleryConverters = 2

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
	client := s.driveClientFor(r.Context())
	if !client.Enabled() {
		writeError(w, 503, "Halaman foto sedang tidak tersedia. Silakan hubungi Elvove.")
		return
	}
	files, err := s.galleryPhotos(r, client, booking)
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
	client := s.driveClientFor(r.Context())
	if booking.RawFolderID == "" || !client.Enabled() {
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
	// Previews live under UPLOAD_DIR, so they survive a redeploy and Drive is
	// read once per photo. The folder is not pruned automatically: delete
	// gallery/<token> once a booking's edit is delivered if the volume gets
	// tight.
	cache := filepath.Join(s.Config.UploadDir, "gallery", token, fileID+".webp")
	// The full-size original is written under "originals", which the uploads
	// handler never serves: otherwise the client could download the untouched
	// photo while the preview is being made.
	original := filepath.Join(s.Config.UploadDir, "originals", "gallery", token, fileID+".source")
	data, err := os.ReadFile(cache)
	if err != nil {
		// One conversion per photo even when the page asks for it twice at once.
		unlock := s.galleryLock(fmt.Sprintf("%d/%s", booking.ID, fileID))
		data, webpID, err = s.convertGalleryPhoto(r.Context(), client, booking, fileID, name, webpID, cache, original)
		unlock()
		if err != nil {
			log.Printf("gallery convert %d/%s (%s): %v", booking.ID, fileID, name, err)
			writeError(w, 502, "Foto tidak dapat dimuat.")
			return
		}
	}
	w.Header().Set("Content-Type", "image/webp")
	// The bytes never change for a given photo, so the browser keeps them.
	w.Header().Set("Cache-Control", "private, max-age=604800")
	w.Write(data)
}

// convertGalleryPhoto makes one preview: the Drive "webp" copy when this server
// has none, the cached file when a previous run already made it, or the original
// otherwise. Drive is asked only once per photo.
func (s *Server) convertGalleryPhoto(ctx context.Context, client *drive.Client, booking galleryBooking, fileID, name, webpID, cache, original string) ([]byte, string, error) {
	// Each conversion decodes a full-size photo. A page asks for dozens at once,
	// and that many cwebp runs together run this server out of memory, so only a
	// couple run at a time.
	select {
	case s.galleryConvert <- struct{}{}:
		defer func() { <-s.galleryConvert }()
	case <-ctx.Done():
		return nil, webpID, ctx.Err()
	}

	// The WebP copy in Drive is a fraction of the original, so it refills a
	// cache that a redeploy or a cleanup removed without converting again.
	if webpID != "" {
		if err := cacheGalleryWebP(ctx, client, webpID, cache); err == nil {
			if data, readErr := os.ReadFile(cache); readErr == nil && len(data) > 0 {
				return data, webpID, nil
			}
		}
	}

	// The preview may exist from a run that failed only at the upload step.
	if data, err := os.ReadFile(cache); err == nil && len(data) > 0 {
		return data, s.uploadGalleryWebP(ctx, client, booking, fileID, name, webpID, data), nil
	}

	body, err := client.Download(ctx, fileID)
	if err != nil {
		return nil, webpID, err
	}
	defer body.Close()
	if err := os.MkdirAll(filepath.Dir(cache), 0750); err != nil {
		return nil, webpID, err
	}
	if err := os.MkdirAll(filepath.Dir(original), 0750); err != nil {
		return nil, webpID, err
	}
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

	// A half-written cache would be served as a broken image forever, so cwebp
	// writes next to it and the file is renamed into place only once complete.
	tmp := cache + ".tmp"
	if err := convertWebP(original, tmp, 1600, 78); err != nil {
		os.Remove(tmp)
		return nil, webpID, err
	}
	if err := os.Rename(tmp, cache); err != nil {
		os.Remove(tmp)
		return nil, webpID, err
	}
	data, err := os.ReadFile(cache)
	if err != nil {
		return nil, webpID, err
	}
	return data, s.uploadGalleryWebP(ctx, client, booking, fileID, name, webpID, data), nil
}

// cacheGalleryWebP copies the preview already stored in Drive to the local
// cache. It is much cheaper than downloading and converting the original again.
func cacheGalleryWebP(ctx context.Context, client *drive.Client, fileID, cache string) error {
	if err := os.MkdirAll(filepath.Dir(cache), 0750); err != nil {
		return err
	}
	body, err := client.Download(ctx, fileID)
	if err != nil {
		return err
	}
	defer body.Close()
	tmp := cache + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, body)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, cache)
}

// uploadGalleryWebP puts a finished preview in the booking's "webp" subfolder
// once. A failure only loses that Drive copy; the page keeps working from the
// local cache and the next run retries.
func (s *Server) uploadGalleryWebP(ctx context.Context, client *drive.Client, booking galleryBooking, fileID, name, webpID string, data []byte) string {
	if webpID != "" {
		return webpID
	}
	// The caller read webp_file_id before taking the per-photo lock; another
	// request may have uploaded the same preview while it waited, so the row wins
	// over that stale empty value and Drive never holds two copies.
	var stored string
	if err := s.Store.Pool.QueryRow(ctx, `SELECT COALESCE(webp_file_id,'') FROM booking_photos WHERE booking_id=$1 AND drive_file_id=$2`, booking.ID, fileID).Scan(&stored); err == nil && stored != "" {
		return stored
	}
	if booking.WebpFolderID != "" {
		// The folder was made with the booking; only a deleted one has to be
		// looked up again.
		if _, folderErr := client.Folder(ctx, booking.WebpFolderID); folderErr != nil {
			booking.WebpFolderID = ""
		}
	}
	folder := booking.WebpFolderID
	if folder == "" {
		var err error
		folder, err = client.EnsureFolder(ctx, booking.RawFolderID, webpFolderName)
		if err != nil {
			log.Printf("gallery webp folder booking %d: %v", booking.ID, err)
			return webpID
		}
	}
	uploaded, err := client.Upload(ctx, folder, strings.TrimSuffix(name, filepath.Ext(name))+".webp", "image/webp", data)
	if err != nil {
		log.Printf("gallery webp upload booking %d: %v", booking.ID, err)
		return webpID
	}
	if _, err := s.Store.Exec(ctx, `UPDATE booking_photos SET webp_file_id=$1,updated_at=NOW() WHERE booking_id=$2 AND drive_file_id=$3`, uploaded, booking.ID, fileID); err != nil {
		log.Printf("gallery webp save booking %d: %v", booking.ID, err)
	}
	return uploaded
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
	// Two tabs saving at once would each compare against the same snapshot and
	// together exceed the quota, so one save runs per booking at a time.
	unlock := s.galleryLock(fmt.Sprintf("select/%d", booking.ID))
	defer unlock()
	// The page sends the whole picked set on every click, so only the rows that
	// actually change are written: one click is one row, not the whole folder.
	want := make(map[string]int, len(req.FileIDs))
	for i, id := range req.FileIDs {
		want[id] = i + 1
	}
	current := map[string]int{}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT drive_file_id, COALESCE(position,0) FROM booking_photos WHERE booking_id=$1 AND selected`, booking.ID)
	if err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
		return
	}
	for rows.Next() {
		var id string
		var position int
		if err := rows.Scan(&id, &position); err == nil {
			current[id] = position
		}
	}
	rows.Close()

	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
		return
	}
	defer tx.Rollback(r.Context())
	for id, position := range current {
		if next, ok := want[id]; ok && next == position {
			continue
		}
		if _, err := tx.Exec(r.Context(), `UPDATE booking_photos SET selected=FALSE,position=NULL,updated_at=NOW() WHERE booking_id=$1 AND drive_file_id=$2`, booking.ID, id); err != nil {
			writeError(w, 500, "Pilihan foto tidak dapat disimpan.")
			return
		}
	}
	for id, position := range want {
		if old, ok := current[id]; ok && old == position {
			continue
		}
		if _, err := tx.Exec(r.Context(), `UPDATE booking_photos SET selected=TRUE,position=$1,updated_at=NOW() WHERE booking_id=$2 AND drive_file_id=$3`, position, booking.ID, id); err != nil {
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

// submitSelection closes the choice: it creates the "foto terpilih" folder beside the
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
	client := s.driveClientFor(r.Context())
	if !client.Enabled() {
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

	folder := booking.MusteditID
	if folder != "" {
		// The folder was made with the booking; only a deleted one is recreated.
		if _, folderErr := client.Folder(r.Context(), folder); folderErr != nil {
			folder = ""
		}
	}
	if folder == "" {
		folder, err = client.EnsureFolder(r.Context(), booking.RawFolderID, pickedFolderName)
	}
	if err != nil {
		log.Printf("gallery picked folder booking %d: %v", booking.ID, err)
		writeError(w, 502, "Folder hasil edit tidak dapat dibuat. Silakan hubungi Elvove.")
		return
	}
	// A retry after a partial failure must not copy the same photo twice, so the
	// folder is read first and names already there are skipped.
	existing, err := client.ListImages(r.Context(), folder)
	if err != nil {
		log.Printf("gallery picked list booking %d: %v", booking.ID, err)
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
	dropped := make([]string, 0)
	copied := 0
	for _, item := range items {
		if already[item.Name] > 0 {
			already[item.Name]--
			copied++
			continue
		}
		if _, err := client.Copy(r.Context(), item.ID, folder, item.Name); err != nil {
			if errors.Is(err, drive.ErrNotFound) {
				// The photographer removed this photo after the client picked
				// it. Drop the pick so the next try is not blocked forever.
				if _, dbErr := s.Store.Exec(r.Context(), `UPDATE booking_photos SET selected=FALSE,updated_at=NOW() WHERE booking_id=$1 AND drive_file_id=$2`, booking.ID, item.ID); dbErr != nil {
					log.Printf("gallery unselect %d/%s: %v", booking.ID, item.ID, dbErr)
				}
				dropped = append(dropped, item.Name)
				continue
			}
			log.Printf("gallery copy %d/%s: %v", booking.ID, item.ID, err)
			writeError(w, 502, "Sebagian foto gagal disalin. Silakan coba kirim lagi.")
			return
		}
		copied++
	}
	if copied == 0 {
		writeError(w, 409, "Foto yang Anda pilih sudah tidak tersedia di folder. Muat ulang halaman ini untuk memilih foto lain.")
		return
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE bookings SET folder_mustedit_id=$1,selection_submitted_at=NOW(),updated_at=NOW() WHERE id=$2`, folder, booking.ID); err != nil {
		log.Printf("gallery submit booking %d: %v", booking.ID, err)
		writeError(w, 500, "Pilihan foto tersimpan, tetapi status gagal diperbarui. Silakan hubungi Elvove.")
		return
	}
	message := "Pilihan foto sudah kami terima. Terima kasih."
	if len(dropped) > 0 {
		message += fmt.Sprintf(" %d foto sudah tidak tersedia di folder sehingga tidak ikut dipilih: %s.", len(dropped), strings.Join(dropped, ", "))
	}
	writeJSON(w, 200, map[string]any{"selectedCount": copied, "submittedAt": wibTimestamp(nowWallClock()), "message": message})
}

// galleryLock serializes the conversion of one photo. The entry is removed
// when nobody holds or waits for it, so the map stays small.
func (s *Server) galleryLock(key string) func() {
	s.galleryMu.Lock()
	lock := s.galleryLocks[key]
	if lock == nil {
		lock = &keyedMutex{}
		s.galleryLocks[key] = lock
	}
	lock.refs++
	s.galleryMu.Unlock()

	lock.Lock()
	return func() {
		lock.Unlock()
		s.galleryMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(s.galleryLocks, key)
		}
		s.galleryMu.Unlock()
	}
}

// keyedMutex is one lock per photo conversion, so two requests for the same
// thumbnail wait instead of both downloading and converting the original.
// refs counts holders and waiters; galleryMu guards it.
type keyedMutex struct {
	mu   sync.Mutex
	refs int
}

func (k *keyedMutex) Lock()   { k.mu.Lock() }
func (k *keyedMutex) Unlock() { k.mu.Unlock() }

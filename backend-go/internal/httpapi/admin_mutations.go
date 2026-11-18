package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	if err := decoder.Decode(target); err != nil {
		writeUnreadableBody(w)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeUnreadableBody(w)
		return false
	}
	return true
}

func decodeNumberJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		writeUnreadableBody(w)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeUnreadableBody(w)
		return false
	}
	return true
}

func validText(value string, max int, required bool) bool {
	trimmed := strings.TrimSpace(value)
	return (!required || trimmed != "") && utf8.RuneCountInString(value) <= max
}

func (s *Server) execAdmin(w http.ResponseWriter, r *http.Request, resource string, id int64, query string, args ...any) (int64, bool) {
	count, err := s.Store.Exec(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database operation failed.")
		return 0, false
	}
	if count == 0 {
		// Java checks existsById first and names the resource it could not find.
		notFoundError(w, resource, id)
		return 0, false
	}
	return count, true
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validText(request.Name, 255, true) {
		writeValidationError(w, map[string]string{"name": "Name is required"})
		return
	}
	// Java's CategoryService.create checks existsByName first and answers 400,
	// not the 409 a unique-constraint failure would produce.
	var exists bool
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM categories WHERE name=$1)`, request.Name).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "Database operation failed.")
		return
	}
	if exists {
		writeError(w, http.StatusBadRequest, "Category '"+request.Name+"' already exists")
		return
	}
	slug := slugify(request.Name)
	row, err := s.Store.QueryOne(r.Context(), `INSERT INTO categories (name, slug, created_at, updated_at) VALUES ($1,$2,NOW(),NOW()) RETURNING id, name, slug`, request.Name, slug)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "Category '"+strings.TrimSpace(request.Name)+"' already exists")
		} else {
			writeError(w, http.StatusInternalServerError, "Database operation failed.")
		}
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (s *Server) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	var request struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validText(request.Name, 255, true) {
		writeValidationError(w, map[string]string{"name": "Name is required"})
		return
	}
	row, err := s.Store.QueryOne(r.Context(), `UPDATE categories SET name=$1, slug=$2, updated_at=NOW() WHERE id=$3 RETURNING id, name, slug`, request.Name, slugify(request.Name), id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, 409, "Category name or slug already exists.")
		} else if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Category", id)
		} else {
			writeError(w, 500, "Category could not be saved.")
		}
		return
	}
	writeJSON(w, 200, row)
}

func (s *Server) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	if _, ok := s.execAdmin(w, r, "Category", id, `DELETE FROM categories WHERE id=$1`, id); ok {
		writeEmpty(w, http.StatusNoContent)
	}
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	dash := false
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

type packageRequest struct {
	Title        string      `json:"title"`
	Price        json.Number `json:"price"`
	Features     []string    `json:"features"`
	DisplayOrder *int        `json:"displayOrder"`
	Highlighted  bool        `json:"highlighted"`
	// PhotoSelectionLimit is how many raw photos the client may pick from this
	// package ("Origin: 10"). Zero means the package includes no selection.
	PhotoSelectionLimit *int `json:"photoSelectionLimit"`
}

func (s *Server) createPackage(w http.ResponseWriter, r *http.Request) { s.savePackage(w, r, 0, true) }
func (s *Server) updatePackage(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	s.savePackage(w, r, id, false)
}
func (s *Server) savePackage(w http.ResponseWriter, r *http.Request, id int64, create bool) {
	var req packageRequest
	if !decodeNumberJSON(w, r, &req) {
		return
	}
	title := strings.TrimSpace(req.Title)
	price, ok := new(big.Rat).SetString(string(req.Price))
	decimalParts := strings.Split(string(req.Price), ".")
	decimalValid := len(decimalParts) == 1 || (len(decimalParts) == 2 && len(decimalParts[1]) <= 2)
	// Mirrors PackageItemRequest: @NotBlank title, @NotNull @Positive price.
	fieldErrors := map[string]string{}
	if !validText(title, 255, true) {
		fieldErrors["title"] = "Title is required"
	}
	if string(req.Price) == "" {
		fieldErrors["price"] = "Price is required"
	} else if !ok || !decimalValid || price.Sign() <= 0 || len(req.Price) > 13 {
		fieldErrors["price"] = "Price must be positive"
	}
	limit := 0
	if req.PhotoSelectionLimit != nil {
		limit = *req.PhotoSelectionLimit
	}
	// A negative count is meaningless; the upper bound keeps a typo from
	// promising a folder-sized selection.
	if limit < 0 || limit > 1000 {
		fieldErrors["photoSelectionLimit"] = "Jumlah foto pilihan harus antara 0 dan 1000"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	order := 0
	if req.DisplayOrder != nil {
		order = *req.DisplayOrder
	}
	features := req.Features
	if features == nil {
		features = []string{}
	}
	for _, feature := range features {
		if !validText(feature, 255, false) {
			writeError(w, 400, "Package feature is too long.")
			return
		}
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "Package could not be saved.")
		return
	}
	defer tx.Rollback(r.Context())
	var pkgID int64
	if create {
		err = tx.QueryRow(r.Context(), `INSERT INTO packages(title,price,display_order,highlighted,photo_selection_limit,created_at,updated_at) VALUES($1,$2,$3,$4,$5,NOW(),NOW()) RETURNING id`, title, string(req.Price), order, req.Highlighted, limit).Scan(&pkgID)
	} else {
		// Java replaces the whole record: a missing displayOrder resets it to
		// 0 (PackageItemService.apply) instead of keeping the previous value.
		err = tx.QueryRow(r.Context(), `UPDATE packages SET title=$1,price=$2,display_order=$3,highlighted=$4,photo_selection_limit=$5,updated_at=NOW() WHERE id=$6 RETURNING id`, title, string(req.Price), order, req.Highlighted, limit, id).Scan(&pkgID)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Package", id)
		} else {
			writeError(w, 500, "Package could not be saved.")
		}
		return
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM package_features WHERE package_id=$1`, pkgID); err == nil {
		for _, feature := range features {
			if _, err = tx.Exec(r.Context(), `INSERT INTO package_features(package_id,feature) VALUES($1,$2)`, pkgID, feature); err != nil {
				break
			}
		}
	}
	if err != nil {
		writeError(w, 500, "Package could not be saved.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Package could not be saved.")
		return
	}
	if !create && req.DisplayOrder == nil {
		_ = s.Store.Pool.QueryRow(r.Context(), `SELECT display_order FROM packages WHERE id=$1`, pkgID).Scan(&order)
	}
	response := map[string]any{"id": pkgID, "title": title, "price": decimalValue(req.Price.String()), "features": features, "displayOrder": order, "highlighted": req.Highlighted, "photoSelectionLimit": limit}
	status := 200
	if create {
		status = 201
	}
	writeJSON(w, status, response)
}

func (s *Server) createPhoto(w http.ResponseWriter, r *http.Request) { s.savePhoto(w, r, 0, true) }
func (s *Server) updatePhoto(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	s.savePhoto(w, r, id, false)
}
func (s *Server) savePhoto(w http.ResponseWriter, r *http.Request, id int64, create bool) {
	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		ImageURL     string `json:"imageUrl"`
		CategoryID   *int64 `json:"categoryId"`
		Featured     bool   `json:"featured"`
		DisplayOrder *int   `json:"displayOrder"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	// Mirrors PhotoRequest: @NotBlank title and imageUrl, @NotNull categoryId.
	fieldErrors := map[string]string{}
	if !validText(req.Title, 255, true) {
		fieldErrors["title"] = "Title is required"
	}
	if strings.TrimSpace(req.ImageURL) == "" {
		fieldErrors["imageUrl"] = "Image URL is required"
	}
	if req.CategoryID == nil || *req.CategoryID < 1 {
		fieldErrors["categoryId"] = "Category is required"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	order := 0
	if req.DisplayOrder != nil {
		order = *req.DisplayOrder
	}
	_, err := s.Store.QueryOne(r.Context(), `SELECT id FROM categories WHERE id=$1`, *req.CategoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Java's PhotoService resolves the category and answers 404.
			notFoundError(w, "Category", *req.CategoryID)
		} else {
			writeError(w, 500, "Photo could not be saved.")
		}
		return
	}
	var result map[string]any
	if create {
		result, err = s.Store.QueryOne(r.Context(), `INSERT INTO photos(title,description,image_url,category_id,featured,display_order,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,NOW(),NOW()) RETURNING id,title,description,image_url AS "imageUrl",featured,display_order AS "displayOrder",category_id AS "categoryId"`, req.Title, req.Description, req.ImageURL, *req.CategoryID, req.Featured, order)
	} else {
		result, err = s.Store.QueryOne(r.Context(), `UPDATE photos SET title=$1,description=$2,image_url=$3,category_id=$4,featured=$5,display_order=COALESCE($6,display_order),updated_at=NOW() WHERE id=$7 RETURNING id,title,description,image_url AS "imageUrl",featured,display_order AS "displayOrder",category_id AS "categoryId"`, req.Title, req.Description, req.ImageURL, *req.CategoryID, req.Featured, req.DisplayOrder, id)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Photo", id)
		} else {
			writeError(w, 500, "Photo could not be saved.")
		}
		return
	}
	if cat, err := s.Store.QueryOne(r.Context(), `SELECT name FROM categories WHERE id=$1`, *req.CategoryID); err == nil {
		result["categoryName"] = cat["name"]
	}
	imageURL, _ := result["imageUrl"].(string)
	thumbnail, preview := s.photoVariants(imageURL)
	result["thumbnailUrl"], result["previewUrl"] = nullableString(thumbnail), nullableString(preview)
	status := 200
	if create {
		status = 201
	}
	writeJSON(w, status, result)
}
func (s *Server) deletePhoto(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	row, err := s.Store.QueryOne(r.Context(), `DELETE FROM photos WHERE id=$1 RETURNING image_url`, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Photo", id)
		} else {
			writeError(w, 500, "Photo could not be deleted.")
		}
		return
	}
	if path, ok := row["image_url"].(string); ok {
		removeUploadVariants(s.Config.UploadDir, path)
	}
	writeEmpty(w, 204)
}
func removeUploadVariants(root, imageURL string) {
	filename := uploadFilename(imageURL)
	if filename == "" {
		return
	}
	deleteUploadFiles(root, strings.TrimSuffix(filename, filepath.Ext(filename)))
	// Legacy upload stored directly below /uploads/.
	_ = os.Remove(filepath.Join(root, filename))
}

// deleteUploadFiles mirrors Java's FileStorageService.deleteById: every
// original with this id (whatever its extension) plus both WebP variants.
func deleteUploadFiles(root, id string) {
	if entries, err := os.ReadDir(filepath.Join(root, "originals")); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), id+".") {
				_ = os.Remove(filepath.Join(root, "originals", entry.Name()))
			}
		}
	}
	_ = os.Remove(filepath.Join(root, "thumbnails", id+".webp"))
	_ = os.Remove(filepath.Join(root, "previews", id+".webp"))
}

func (s *Server) getNotificationSettings(w http.ResponseWriter, r *http.Request) {
	enabled := false
	_ = s.Store.Pool.QueryRow(r.Context(), `SELECT enabled FROM booking_notification_settings WHERE id=1`).Scan(&enabled)
	recipients, err := s.Store.QueryRows(r.Context(), `SELECT email FROM booking_notification_recipients ORDER BY id`)
	if err != nil {
		writeError(w, 500, "Notification settings could not be loaded.")
		return
	}
	emails := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		if value, ok := recipient["email"].(string); ok {
			emails = append(emails, value)
		}
	}
	writeJSON(w, 200, map[string]any{"enabled": enabled, "recipients": emails})
}
func (s *Server) saveNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled    *bool    `json:"enabled"`
		Recipients []string `json:"recipients"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	// Java's SaveRequest marks both fields @NotNull: an absent field is a
	// validation error naming that field, not a generic bad request. Sending an
	// empty array is still allowed and clears the list deliberately.
	fieldErrors := map[string]string{}
	if req.Enabled == nil {
		fieldErrors["enabled"] = "Status notifikasi wajib diisi."
	}
	if req.Recipients == nil {
		fieldErrors["recipients"] = "Daftar penerima wajib diisi. Kirim array kosong untuk mengosongkannya."
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	if len(req.Recipients) > 20 {
		writeError(w, 400, "Maksimal 20 alamat email penerima.")
		return
	}
	emails := make([]string, 0, len(req.Recipients))
	seen := map[string]bool{}
	emailPattern := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	for _, raw := range req.Recipients {
		email := strings.TrimSpace(raw)
		if email == "" {
			writeError(w, 400, "Alamat email penerima tidak boleh kosong.")
			return
		}
		if len(email) > 255 || !emailPattern.MatchString(email) {
			writeError(w, 400, "Format email tidak valid: "+email)
			return
		}
		key := strings.ToLower(email)
		if seen[key] {
			writeError(w, 400, "Email penerima duplikat: "+email)
			return
		}
		seen[key] = true
		emails = append(emails, email)
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "Notification settings could not be saved.")
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `DELETE FROM booking_notification_recipients`); err == nil {
		for _, email := range emails {
			if _, err = tx.Exec(r.Context(), `INSERT INTO booking_notification_recipients(email,created_at) VALUES($1,NOW())`, email); err != nil {
				break
			}
		}
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO booking_notification_settings(id,enabled,updated_at) VALUES(1,$1,NOW()) ON CONFLICT(id) DO UPDATE SET enabled=EXCLUDED.enabled,updated_at=NOW()`, *req.Enabled)
	}
	if err != nil {
		writeError(w, 400, "Email penerima duplikat atau tidak valid. Periksa kembali daftar penerima.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Notification settings could not be saved.")
		return
	}
	writeJSON(w, 200, map[string]any{"enabled": *req.Enabled, "recipients": emails})
}

type smtpConfig struct {
	Host              string
	Port              int
	Username          string
	EncryptedPassword string
	Encryption        string
	Authentication    bool
	SenderName        string
	SenderEmail       string
}

// bookingIDFromPath reads and validates the {id} path value shared by the
// booking mutation endpoints.
func bookingIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return 0, false
	}
	return id, true
}

func (s *Server) sendBookingConfirmation(w http.ResponseWriter, r *http.Request) {
	s.sendCustomerMail(w, r, customerMail{
		Label:       "konfirmasi",
		NoEmail:     "Pemesanan ini tidak memiliki alamat email klien, sehingga konfirmasi tidak dapat dikirim.",
		SentColumn:  "confirmation_email_sent_at",
		CountColumn: "confirmation_email_count",
		SentKey:     "confirmationEmailSentAt",
		CountKey:    "confirmationEmailCount",
		Refuse: func(b mailBooking) string {
			if b.Status != "ACCEPTED" {
				return "Email konfirmasi hanya dapat dikirim untuk pemesanan yang sudah diterima."
			}
			return ""
		},
		Render: func(b mailBooking, sender, whatsapp, siteURL string) (string, string, string) {
			return renderBookingConfirmation(b.bookingEmail, sender, whatsapp, siteURL)
		},
	})
}

// sendBookingInvoice closes the order: the transaction as paid, plus the link to
// the edited photos. It is refused until the booking is finished and paid and
// the Drive folder is filled in, because a final invoice with no gallery or with
// an open balance would be wrong.
func (s *Server) sendBookingInvoice(w http.ResponseWriter, r *http.Request) {
	s.sendCustomerMail(w, r, customerMail{
		Label:       "invoice",
		NoEmail:     "Pemesanan ini tidak memiliki alamat email klien, sehingga invoice tidak dapat dikirim.",
		SentColumn:  "invoice_email_sent_at",
		CountColumn: "invoice_email_count",
		SentKey:     "invoiceEmailSentAt",
		CountKey:    "invoiceEmailCount",
		Refuse: func(b mailBooking) string {
			if b.Status != "COMPLETED" {
				return "Invoice hanya dapat dikirim setelah pemesanan selesai."
			}
			if b.SettledAt == "" {
				return "Tandai pembayaran sebagai lunas terlebih dahulu, lalu kirim invoice."
			}
			if strings.TrimSpace(b.DeliveryURL) == "" {
				return "Link folder foto edit belum ada. Buka detail pemesanan, tekan Periksa pada folder foto klien agar tautannya terisi, lalu kirim invoice."
			}
			return ""
		},
		Render: func(b mailBooking, sender, whatsapp, siteURL string) (string, string, string) {
			return renderBookingInvoice(b.bookingEmail, b.invoiceDetails(nowWallClock()), sender, whatsapp, siteURL)
		},
	})
}

// sendBookingSettlement sends the reminder for the remaining half. A booking
// marked lunas has nothing left to settle, so it is refused rather than sent a
// message asking for money that already arrived.
func (s *Server) sendBookingSettlement(w http.ResponseWriter, r *http.Request) {
	s.sendCustomerMail(w, r, customerMail{
		Label:       "pelunasan",
		NoEmail:     "Pemesanan ini tidak memiliki alamat email klien, sehingga email pelunasan tidak dapat dikirim.",
		SentColumn:  "settlement_email_sent_at",
		CountColumn: "settlement_email_count",
		SentKey:     "settlementEmailSentAt",
		CountKey:    "settlementEmailCount",
		Refuse: func(b mailBooking) string {
			if b.Status != "ACCEPTED" && b.Status != "COMPLETED" {
				return "Email pelunasan hanya dapat dikirim untuk pemesanan yang sudah diterima."
			}
			if b.SettledAt != "" {
				return "Pemesanan ini sudah ditandai lunas, sehingga tidak ada sisa pembayaran yang perlu ditagih."
			}
			// The notice states the down payment has arrived. Sending it before
			// that is recorded would tell the customer something untrue, so the
			// admin marks the DP first.
			if b.DownPaidAt == "" {
				return "Tandai DP sebagai dibayar terlebih dahulu, lalu kirim email pelunasan."
			}
			return ""
		},
		Render: func(b mailBooking, sender, whatsapp, siteURL string) (string, string, string) {
			return renderBookingSettlement(b.bookingEmail, b.invoiceDetails(nowWallClock()), sender, whatsapp, siteURL)
		},
	})
}

// sendBookingSelection sends the notice that opens the client page. It goes out
// before the settlement one, so the client picks the photos while they are
// fresh and the settlement notice stays a request for money only.
func (s *Server) sendBookingSelection(w http.ResponseWriter, r *http.Request) {
	s.sendCustomerMail(w, r, customerMail{
		Label:       "pilihan foto",
		NoEmail:     "Pemesanan ini tidak memiliki alamat email klien, sehingga email pilihan foto tidak dapat dikirim.",
		SentColumn:  "selection_email_sent_at",
		CountColumn: "selection_email_count",
		SentKey:     "selectionEmailSentAt",
		CountKey:    "selectionEmailCount",
		Refuse: func(b mailBooking) string {
			if b.Status != "ACCEPTED" && b.Status != "COMPLETED" {
				return "Email pilihan foto hanya dapat dikirim untuk pemesanan yang sudah diterima."
			}
			if b.RawFolder == "" {
				return "Folder foto klien belum ada. Buka detail pemesanan, tekan Periksa pada folder foto klien, lalu kirim email pilihan foto."
			}
			return ""
		},
		Render: func(b mailBooking, sender, whatsapp, siteURL string) (string, string, string) {
			// A booking taken before selection links existed has no token, and a
			// notice without the link cannot be acted on.
			if b.SelectionToken == "" {
				b.SelectionToken = s.ensureSelectionToken(r.Context(), b.ID)
			}
			details := b.invoiceDetails(nowWallClock())
			details.GalleryURL = b.selectionLink(siteURL)
			return renderBookingSelection(b.bookingEmail, details, sender, whatsapp, siteURL)
		},
	})
}

// updateBookingPayment records where the payment stands: nothing, the down
// payment, or the whole amount. The request carries the state and the two
// timestamps follow from it, so a booking can never claim "lunas" without a
// down-payment date.
func (s *Server) updateBookingPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := bookingIDFromPath(w, r)
	if !ok {
		return
	}
	var req struct {
		PaymentStatus string `json:"paymentStatus"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.PaymentStatus == "" {
		writeValidationError(w, map[string]string{"paymentStatus": "Status pembayaran wajib diisi."})
		return
	}
	var update string
	switch req.PaymentStatus {
	case "UNPAID":
		update = `UPDATE bookings SET down_payment_paid_at=NULL,settlement_paid_at=NULL,updated_at=NOW() WHERE id=$1`
	case "DOWN_PAYMENT_PAID":
		// The DP time is kept on a resend of the same state, so the recorded
		// moment stays the one the admin first confirmed.
		update = `UPDATE bookings SET down_payment_paid_at=COALESCE(down_payment_paid_at,NOW()),settlement_paid_at=NULL,updated_at=NOW() WHERE id=$1`
	case "PAID":
		update = `UPDATE bookings SET down_payment_paid_at=COALESCE(down_payment_paid_at,NOW()),settlement_paid_at=COALESCE(settlement_paid_at,NOW()),updated_at=NOW() WHERE id=$1`
	default:
		// An unknown value is a body the Java DTO would refuse to parse, the same
		// split PATCH /status makes for an unknown status.
		writeUnreadableBody(w)
		return
	}
	if _, err := s.Store.Exec(r.Context(), update, id); err != nil {
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

// updateBookingDelivery stores the Google Drive folder holding the edited
// photos. The invoice e-mail carries it, so the admin pastes it once here
// instead of writing it into every message by hand. An empty value clears it.
func (s *Server) updateBookingDelivery(w http.ResponseWriter, r *http.Request) {
	id, ok := bookingIDFromPath(w, r)
	if !ok {
		return
	}
	var req struct {
		DeliveryURL *string `json:"deliveryUrl"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.DeliveryURL == nil {
		writeValidationError(w, map[string]string{"deliveryUrl": "Link foto wajib diisi."})
		return
	}
	link := strings.TrimSpace(*req.DeliveryURL)
	if link != "" {
		parsed, err := url.Parse(link)
		// A mail button needs http(s); anything else (javascript:, a bare
		// string) would render as a dead or unsafe link in the message.
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			writeValidationError(w, map[string]string{"deliveryUrl": "Link foto harus berupa URL http(s) yang valid."})
			return
		}
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE bookings SET delivery_url=NULLIF($2,''),updated_at=NOW() WHERE id=$1`, id, link); err != nil {
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

// mailSenderName reads the configured sender name used in the e-mail
// templates, mirroring Java's SmtpMailService.getSenderName(). It falls back to
// the brand name when SMTP is not configured yet, so a template still renders.
func (s *Server) mailSenderName(r *http.Request) string {
	settings, err := s.loadSMTP(r)
	if err != nil || strings.TrimSpace(settings.SenderName) == "" {
		return emailBrandFallback
	}
	return settings.SenderName
}

// contactWhatsApp reads the number from the site's contact settings, the one
// the public WhatsApp button uses. Empty when it has not been filled in.
func (s *Server) contactWhatsApp(r *http.Request) string {
	var number string
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT COALESCE(whatsapp_number,'') FROM contact_info ORDER BY id LIMIT 1`).Scan(&number); err != nil {
		log.Printf("whatsapp number unavailable for e-mail: %v", err)
		return ""
	}
	return strings.TrimSpace(number)
}
func (s *Server) notifyBookingAdmins(id int64) {
	// The goroutine outlives the handler, so a panic here would take the whole
	// process down: the HTTP recover middleware cannot catch it.
	defer func() {
		if v := recover(); v != nil {
			log.Printf("Notifikasi booking #%d panik: %v", id, v)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var enabled bool
	if err := s.Store.Pool.QueryRow(ctx, `SELECT enabled FROM booking_notification_settings WHERE id=1`).Scan(&enabled); err != nil {
		log.Printf("Notifikasi booking #%d dilewati: pengaturan notifikasi tidak terbaca: %v", id, err)
		return
	}
	if !enabled {
		return
	}
	recipients, err := s.Store.QueryRows(ctx, `SELECT email FROM booking_notification_recipients ORDER BY id`)
	if err != nil {
		log.Printf("Notifikasi booking #%d dilewati: daftar penerima tidak terbaca: %v", id, err)
		return
	}
	if len(recipients) == 0 {
		// Java logs the same warning; without it an empty list looks like a bug.
		log.Printf("Notifikasi booking #%d dilewati: notifikasi aktif tetapi daftar penerima kosong", id)
		return
	}
	b := bookingEmail{ID: id}
	if err = s.Store.Pool.QueryRow(ctx, `SELECT name,email,phone,to_char(event_date,'YYYY-MM-DD'),COALESCE(to_char(event_time,'HH24:MI'),''),COALESCE(event_location,''),COALESCE(package_title,''),COALESCE(package_price::text,''),status FROM bookings WHERE id=$1`, id).
		Scan(&b.Name, &b.Email, &b.Phone, &b.EventDate, &b.EventTime, &b.EventLocation, &b.PackageTitle, &b.PackagePrice, &b.Status); err != nil {
		log.Printf("Notifikasi booking #%d dilewati: data booking tidak terbaca: %v", id, err)
		return
	}
	link, err := adminBookingURL(s.Config.MailAdminURL, id)
	if err != nil {
		log.Printf("admin booking link for booking %d is unavailable: %v", id, err)
		return
	}
	// A bare request keeps these lookups off the customer's context: the
	// goroutine outlives the handler that started it.
	mailReq := &http.Request{}
	subject, htmlBody, textBody := renderAdminNotification(b, s.mailSenderName(mailReq), link, s.Config.SiteURL)
	sent := 0
	for _, recipient := range recipients {
		email, _ := recipient["email"].(string)
		if err := sendWithRetry(func() error {
			return s.sendSMTPMessage(mailReq, email, subject, htmlBody, textBody)
		}); err != nil {
			// One recipient failing must not stop the others. The address is
			// left out of the log; the cause is kept so the failure is
			// diagnosable instead of silent.
			log.Printf("Notifikasi booking #%d gagal dikirim ke satu penerima: %v", id, err)
			continue
		}
		sent++
	}
	// Without a line on success, "no notification log" cannot be told apart
	// from "notification never ran".
	log.Printf("Notifikasi booking #%d terkirim ke %d/%d penerima", id, sent, len(recipients))
}

// retryBackoff is a variable so tests do not wait out the real delay.
var retryBackoff = 2 * time.Second

// sendWithRetry retries a transient SMTP failure a few times. Gmail answers 421
// and drops idle connections, so a single attempt loses the notification for
// good. A rejection of the address itself is not retried.
func sendWithRetry(send func() error) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if err = send(); err == nil {
			return nil
		}
		if isPermanentSMTPError(err) {
			return err
		}
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * retryBackoff)
		}
	}
	return err
}

// isPermanentSMTPError reports a 5xx answer: the server has already refused
// the message, so sending it again only repeats the same rejection.
func isPermanentSMTPError(err error) bool {
	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		return protoErr.Code >= 500
	}
	return false
}

// adminBookingURL mirrors BookingLinks.adminBookingUrl: the configured base is
// checked so a credential can never end up in an e-mail.
func adminBookingURL(baseURL string, bookingID int64) (string, error) {
	if bookingID <= 0 {
		return "", fmt.Errorf("ID booking tidak valid.")
	}
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("app.mail.admin-url belum diatur.")
	}
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("app.mail.admin-url tidak valid.")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("app.mail.admin-url harus berupa URL http(s) tanpa kredensial.")
	}
	return trimmed + "/admin/bookings?booking=" + strconv.FormatInt(bookingID, 10), nil
}
func (s *Server) deletePackage(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	if _, ok := s.execAdmin(w, r, "Package", id, `DELETE FROM packages WHERE id=$1`, id); ok {
		writeEmpty(w, 204)
	}
}

func (s *Server) upsertSingleton(w http.ResponseWriter, r *http.Request, table, insertSQL, updateSQL, selectSQL string, args []any) (map[string]any, error) {
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(r.Context())
	id := int64(0)
	err = tx.QueryRow(r.Context(), `SELECT id FROM `+table+` ORDER BY id LIMIT 1 FOR UPDATE`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		tag, e := tx.Exec(r.Context(), insertSQL, args...)
		if e != nil {
			return nil, e
		}
		if tag.RowsAffected() == 1 {
			if e = tx.Commit(r.Context()); e != nil {
				return nil, e
			}
			return s.Store.QueryOne(r.Context(), selectSQL)
		}
		err = tx.QueryRow(r.Context(), `SELECT id FROM `+table+` ORDER BY id LIMIT 1 FOR UPDATE`).Scan(&id)
	}
	if err != nil {
		return nil, err
	}
	updateArgs := append(append([]any{}, args...), id)
	if _, err = tx.Exec(r.Context(), updateSQL, updateArgs...); err != nil {
		return nil, err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return nil, err
	}
	return s.Store.QueryOne(r.Context(), selectSQL)
}

func (s *Server) updateAbout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Introduction string `json:"introduction"`
		Years        *int   `json:"yearsOfExperience"`
		Style        string `json:"photographyStyle"`
		Achievements string `json:"achievements"`
		Story        string `json:"story"`
		ImageURL     string `json:"imageUrl"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Years != nil && *req.Years < 0 {
		writeValidationError(w, map[string]string{"yearsOfExperience": "Years of experience cannot be negative"})
		return
	}
	var years any
	if req.Years != nil {
		years = *req.Years
	}
	row, err := s.upsertSingleton(w, r, "about_info", `INSERT INTO about_info(introduction,years_of_experience,photography_style,achievements,story,image_url,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,NOW(),NOW())`, `UPDATE about_info SET introduction=$1,years_of_experience=$2,photography_style=$3,achievements=$4,story=$5,image_url=$6,updated_at=NOW() WHERE id=$7 RETURNING id,introduction,years_of_experience AS "yearsOfExperience",photography_style AS "photographyStyle",achievements,story,image_url AS "imageUrl"`, `SELECT id,introduction,years_of_experience AS "yearsOfExperience",photography_style AS "photographyStyle",achievements,story,image_url AS "imageUrl" FROM about_info ORDER BY id LIMIT 1`, []any{req.Introduction, years, req.Style, req.Achievements, req.Story, req.ImageURL})
	if err != nil {
		writeError(w, 500, "About content could not be saved.")
		return
	}
	writeJSON(w, 200, row)
}
func (s *Server) updateContact(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Address  string `json:"address"`
		Map      string `json:"googleMapLink"`
		WhatsApp string `json:"whatsappNumber"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	row, err := s.upsertSingleton(w, r, "contact_info", `INSERT INTO contact_info(phone,email,address,google_map_link,whatsapp_number,created_at,updated_at) VALUES($1,$2,$3,$4,$5,NOW(),NOW())`, `UPDATE contact_info SET phone=$1,email=$2,address=$3,google_map_link=$4,whatsapp_number=$5,updated_at=NOW() WHERE id=$6 RETURNING id,phone,email,address,google_map_link AS "googleMapLink",whatsapp_number AS "whatsappNumber"`, `SELECT id,phone,email,address,google_map_link AS "googleMapLink",whatsapp_number AS "whatsappNumber" FROM contact_info ORDER BY id LIMIT 1`, []any{req.Phone, req.Email, req.Address, req.Map, req.WhatsApp})
	if err != nil {
		writeError(w, 500, "Contact details could not be saved.")
		return
	}
	writeJSON(w, 200, row)
}

// socialLinkPattern is SocialLinksRequest's @Pattern, verbatim.
var socialLinkPattern = regexp.MustCompile(`^$|^https?://.*`)

func (s *Server) updateSocialLinks(w http.ResponseWriter, r *http.Request) {
	var req map[string]*string
	if !decodeJSON(w, r, &req) {
		return
	}
	names := []string{"instagram", "threads", "tiktok", "facebook", "youtube", "pinterest", "linkedin"}
	values := make([]any, 0, 7)
	// Mirrors SocialLinksRequest's @Pattern("^$|^https?://.*"): every link is
	// either empty or an http(s) URL. The value is stored exactly as sent.
	fieldErrors := map[string]string{}
	for _, name := range names {
		raw := ""
		if req[name] != nil {
			raw = *req[name]
		}
		if !socialLinkPattern.MatchString(raw) {
			fieldErrors[name] = strings.ToUpper(name[:1]) + name[1:] + " link must be a valid URL"
		}
		values = append(values, optionalText(req[name]))
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	row, err := s.upsertSingleton(w, r, "social_links", `INSERT INTO social_links(instagram,threads,tiktok,facebook,youtube,pinterest,linkedin,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())`, `UPDATE social_links SET instagram=$1,threads=$2,tiktok=$3,facebook=$4,youtube=$5,pinterest=$6,linkedin=$7,updated_at=NOW() WHERE id=$8 RETURNING id,instagram,threads,tiktok,facebook,youtube,pinterest,linkedin`, `SELECT id,instagram,threads,tiktok,facebook,youtube,pinterest,linkedin FROM social_links ORDER BY id LIMIT 1`, values)
	if err != nil {
		writeError(w, 500, "Social links could not be saved.")
		return
	}
	writeJSON(w, 200, row)
}
func (s *Server) updateTerms(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		// Nil keeps the stored introduction: the admin page does not edit it.
		Introduction  *string `json:"introduction"`
		Content       string  `json:"content"`
		LayoutColumns int     `json:"layoutColumns"`
		LayoutWidth   string  `json:"layoutWidth"`
		LayoutSplit   string  `json:"layoutSplit"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	// Mirrors TermsRequest: @NotBlank title and content (no message on the DTO).
	fieldErrors := map[string]string{}
	if strings.TrimSpace(req.Title) == "" {
		fieldErrors["title"] = "must not be blank"
	}
	if strings.TrimSpace(req.Content) == "" {
		fieldErrors["content"] = "must not be blank"
	}
	// The layout values come from three fixed choices in the admin page.
	if req.LayoutColumns != 1 && req.LayoutColumns != 2 {
		fieldErrors["layoutColumns"] = "must be 1 or 2"
	}
	if req.LayoutWidth != "equal" && req.LayoutWidth != "wide-left" && req.LayoutWidth != "wide-right" {
		fieldErrors["layoutWidth"] = "must be equal, wide-left or wide-right"
	}
	if req.LayoutSplit != "heading" && req.LayoutSplit != "marker" {
		fieldErrors["layoutSplit"] = "must be heading or marker"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	tx, e := s.Store.Pool.Begin(r.Context())
	if e != nil {
		writeError(w, 500, "Terms could not be saved.")
		return
	}
	defer tx.Rollback(r.Context())
	var id int64
	var title, introduction, content, version string
	var layoutColumns int
	var layoutWidth, layoutSplit string
	var updated time.Time
	err := tx.QueryRow(r.Context(), `SELECT id,title,COALESCE(introduction,''),content,version,layout_columns,layout_width,layout_split,updated_at FROM site_terms ORDER BY id LIMIT 1 FOR UPDATE`).Scan(&id, &title, &introduction, &content, &version, &layoutColumns, &layoutWidth, &layoutSplit, &updated)
	if req.Introduction == nil {
		req.Introduction = &introduction
	}
	if errors.Is(err, pgx.ErrNoRows) {
		version = "1"
		err = tx.QueryRow(r.Context(), `INSERT INTO site_terms(title,introduction,content,version,layout_columns,layout_width,layout_split,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,NOW(),NOW()) RETURNING id,title,introduction,content,version,layout_columns,layout_width,layout_split,updated_at`, req.Title, *req.Introduction, req.Content, version, req.LayoutColumns, req.LayoutWidth, req.LayoutSplit).Scan(&id, &title, &introduction, &content, &version, &layoutColumns, &layoutWidth, &layoutSplit, &updated)
	} else if err == nil {
		// A layout change is not a content change, so it keeps the terms version
		// the bookings accepted.
		if title != req.Title || deref(&introduction) != *req.Introduction || content != req.Content {
			n, _ := strconv.Atoi(version)
			version = strconv.Itoa(n + 1)
		}
		err = tx.QueryRow(r.Context(), `UPDATE site_terms SET title=$1,introduction=$2,content=$3,version=$4,layout_columns=$5,layout_width=$6,layout_split=$7,updated_at=NOW() WHERE id=$8 RETURNING id,title,introduction,content,version,layout_columns,layout_width,layout_split,updated_at`, req.Title, *req.Introduction, req.Content, version, req.LayoutColumns, req.LayoutWidth, req.LayoutSplit, id).Scan(&id, &title, &introduction, &content, &version, &layoutColumns, &layoutWidth, &layoutSplit, &updated)
	}
	if err != nil {
		writeError(w, 500, "Terms could not be saved.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Terms could not be saved.")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "title": title, "introduction": introduction, "content": content, "version": version, "layoutColumns": layoutColumns, "layoutWidth": layoutWidth, "layoutSplit": layoutSplit, "updatedAt": updated})
}

func (s *Server) updateBookingStatus(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	// BookingStatusUpdateRequest.status is a BookingStatus, so Jackson rejects an
	// unknown constant (and a lowercase one) while deserializing the body, which
	// surfaces as the generic unreadable-body 400 rather than a fieldError.
	if req.Status == "" {
		writeValidationError(w, map[string]string{"status": "Status pemesanan wajib diisi."})
		return
	}
	if !validStatus(req.Status) {
		writeUnreadableBody(w)
		return
	}
	s.transitionBooking(w, r, id, req.Status, false)
}
func (s *Server) acceptBooking(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	if !s.transitionBooking(w, r, id, "ACCEPTED", true) {
		return
	}
}
func validStatus(status string) bool {
	return status == "PENDING" || status == "ACCEPTED" || status == "RESCHEDULE_REQUESTED" || status == "COMPLETED"
}

// statusLabels mirrors Java's BookingService.STATUS_LABELS, used in the error
// messages the admin panel shows.
var statusLabels = map[string]string{
	"PENDING":              "Menunggu",
	"ACCEPTED":             "Diterima",
	"RESCHEDULE_REQUESTED": "Minta Ubah Jadwal",
	"COMPLETED":            "Selesai",
}

func statusLabel(status string) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return status
}

// transitionBooking mirrors BookingService.validateTransition: a 400 in
// Indonesian that distinguishes "already in that status" from "not allowed".
func (s *Server) transitionBooking(w http.ResponseWriter, r *http.Request, id int64, target string, accept bool) bool {
	var current string
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT status FROM bookings WHERE id=$1`, id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		notFoundError(w, "Booking", id)
		return false
	}
	if err != nil {
		writeError(w, 500, "Booking could not be updated.")
		return false
	}
	if current == target {
		writeError(w, 400, `Pemesanan sudah berstatus "`+statusLabel(current)+`".`)
		return false
	}
	// The schedule is locked once the customer has agreed to it: an ACCEPTED
	// booking only moves forward to COMPLETED, and a COMPLETED one does not move
	// at all. A reschedule has to start while the booking is still waiting for a
	// decision, and once a new schedule is proposed the booking must pass through
	// ACCEPTED again before it can be completed.
	allowed := map[string][]string{"PENDING": {"ACCEPTED", "RESCHEDULE_REQUESTED"}, "ACCEPTED": {"COMPLETED"}, "RESCHEDULE_REQUESTED": {"ACCEPTED"}}
	valid := false
	for _, next := range allowed[current] {
		if next == target {
			valid = true
		}
	}
	if !valid {
		writeError(w, 400, fmt.Sprintf(`Status pemesanan tidak dapat diubah dari "%s" menjadi "%s".`, statusLabel(current), statusLabel(target)))
		return false
	}
	// The compare-and-set keeps a status another admin changed a moment ago from
	// being overwritten: only the transition read above may be written.
	affected, err := s.Store.Exec(r.Context(), `UPDATE bookings SET status=$1,updated_at=NOW() WHERE id=$2 AND status=$3`, target, id, current)
	if err != nil {
		writeError(w, 500, "Booking could not be updated.")
		return false
	}
	if affected == 0 {
		writeError(w, 409, "Status pemesanan baru saja diubah. Muat ulang daftar lalu coba lagi.")
		return false
	}
	booking, err := s.fetchBooking(r, id)
	if err != nil {
		writeError(w, 500, "Booking could not be updated.")
		return false
	}
	if accept {
		// Java's accept handler answers with the message plus the new status.
		writeJSON(w, 200, map[string]any{"message": "Pemesanan ditandai sebagai diterima. Email konfirmasi belum dikirim.", "status": target})
	} else {
		// Java's PATCH /status answers with the full BookingResponse.
		writeJSON(w, 200, booking)
	}
	return true
}

// fetchBooking loads one booking in the same shape getBookings returns, so a
// single-record response matches the list contract.
func (s *Server) fetchBooking(r *http.Request, id int64) (map[string]any, error) {
	rows, err := s.Store.QueryRows(r.Context(), bookingDetailColumns+` WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, pgx.ErrNoRows
	}
	normalizeBooking(rows[0])
	return rows[0], nil
}
func (s *Server) rescheduleBooking(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	var req struct {
		EventDate *string `json:"eventDate"`
		EventTime *string `json:"eventTime"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	// Mirrors RescheduleRequest: @NotNull on both fields, but a value Jackson
	// cannot parse (an empty or invalid date) is an unreadable body, not a
	// fieldError — the same split PATCH /status makes for an unknown enum.
	fieldErrors := map[string]string{}
	var date, clock time.Time
	if req.EventDate == nil {
		fieldErrors["eventDate"] = "must not be null"
	} else if parsed, err := time.Parse("2006-01-02", *req.EventDate); err != nil {
		writeUnreadableBody(w)
		return
	} else {
		date = parsed
	}
	if req.EventTime == nil {
		fieldErrors["eventTime"] = "must not be null"
	} else if clock, e = parseLocalTime(*req.EventTime); e != nil {
		writeUnreadableBody(w)
		return
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	// Mirrors Java's @Transactional reschedule + sendQuietly: the row change and
	// the notification e-mail succeed or fail together — a mail error rolls the
	// update back and surfaces as a 400 (BookingWorkflowController.reschedule).
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "Booking could not be rescheduled.")
		return
	}
	defer tx.Rollback(r.Context())
	var current string
	b := bookingEmail{ID: id, EventDate: date.Format("2006-01-02"), EventTime: clock.Format("15:04")}
	if err := tx.QueryRow(r.Context(), `SELECT status, email, name, COALESCE(package_title,''), COALESCE(package_price::text,''), COALESCE(event_location,'') FROM bookings WHERE id=$1 FOR UPDATE`, id).Scan(&current, &b.Email, &b.Name, &b.PackageTitle, &b.PackagePrice, &b.EventLocation); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "Pemesanan tidak ditemukan.")
		return
	} else if err != nil {
		writeError(w, 500, "Booking could not be rescheduled.")
		return
	}
	// The agreed schedule no longer moves: only a booking still waiting for a
	// decision (PENDING) or for the customer to agree to a new date
	// (RESCHEDULE_REQUESTED) may be rescheduled. ACCEPTED and COMPLETED are
	// refused here as well, so the rule holds even if the admin panel is bypassed.
	if current != "PENDING" && current != "RESCHEDULE_REQUESTED" {
		writeError(w, 400, fmt.Sprintf(`Status pemesanan tidak dapat diubah dari "%s" menjadi "%s".`, statusLabel(current), statusLabel("RESCHEDULE_REQUESTED")))
		return
	}
	var newStatus string
	if err := tx.QueryRow(r.Context(), `UPDATE bookings SET event_date=$1,event_time=$2,status='RESCHEDULE_REQUESTED',updated_at=NOW() WHERE id=$3 RETURNING status`, date, clock.Format("15:04:05"), id).Scan(&newStatus); err != nil {
		writeError(w, 500, "Booking could not be rescheduled.")
		return
	}
	subject, htmlBody, textBody := renderBookingReschedule(b, s.mailSenderName(r), s.contactWhatsApp(r), s.Config.SiteURL)
	if mailErr := s.sendSMTPMessage(r, b.Email, subject, htmlBody, textBody); mailErr != nil {
		// Returning without Commit rolls the schedule change back, exactly
		// like Java's BadRequestException inside the transaction.
		log.Printf("reschedule mail failed for booking %d: %v", id, mailErr)
		writeError(w, 400, "Jadwal tersimpan, tetapi email pemberitahuan jadwal gagal dikirim. Periksa Pengaturan Email.")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Booking could not be rescheduled.")
		return
	}
	writeJSON(w, 200, map[string]any{"message": "Jadwal diperbarui dan email pemberitahuan jadwal telah dikirim.", "status": newStatus})
}

// parseLocalTime accepts the ISO local-time forms Jackson accepts.
func parseLocalTime(value string) (time.Time, error) {
	for _, layout := range []string{"15:04:05.999999999", "15:04:05", "15:04"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid local time %q", value)
}
func (s *Server) deleteBooking(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	// Read what has to be cleaned up before the row goes: afterwards there is
	// no folder id or token left to name it.
	var folderID, token string
	if err := s.Store.Pool.QueryRow(r.Context(),
		`SELECT COALESCE(raw_folder_id,''), COALESCE(selection_token,'')
			FROM bookings WHERE id=$1`, id).Scan(&folderID, &token); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "Database operation failed.")
		return
	}
	if _, ok := s.execAdmin(w, r, "Booking", id, `DELETE FROM bookings WHERE id=$1`, id); ok {
		// The folder goes to the trash rather than being destroyed, so a booking
		// removed by mistake does not take the client's photos with it.
		s.trashBookingFolder(r.Context(), folderID)
		if token != "" {
			os.RemoveAll(filepath.Join(s.Config.UploadDir, "gallery", token))
		}
		writeEmpty(w, 204)
	}
}

// trashBookingFolder moves a booking's Drive folder to the trash. A Drive that
// is not connected, or a folder somebody already removed, is not an error: the
// booking is gone either way, and the log keeps the reason.
func (s *Server) trashBookingFolder(ctx context.Context, folderID string) {
	if folderID == "" {
		return
	}
	client := s.driveClientFor(ctx)
	if !client.Enabled() {
		return
	}
	if err := client.Trash(ctx, folderID); err != nil {
		log.Printf("drive trash booking folder %s: %v", folderID, err)
	}
}

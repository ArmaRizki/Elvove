package httpapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
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
		writeError(w, http.StatusConflict, "Category '"+strings.TrimSpace(request.Name)+"' already exists")
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
		if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Category", id)
		} else {
			writeError(w, 409, "Category name or slug already exists.")
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

func (s *Server) createFAQ(w http.ResponseWriter, r *http.Request) { s.saveFAQ(w, r, 0, true) }
func (s *Server) updateFAQ(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	s.saveFAQ(w, r, id, false)
}
func (s *Server) saveFAQ(w http.ResponseWriter, r *http.Request, id int64, create bool) {
	var req struct {
		Question     string `json:"question"`
		Answer       string `json:"answer"`
		DisplayOrder *int   `json:"displayOrder"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	// Mirrors FaqRequest: @NotBlank on question and answer.
	fieldErrors := map[string]string{}
	if req.Question == "" || !validText(req.Question, 255, true) {
		fieldErrors["question"] = "Question is required"
	}
	if strings.TrimSpace(req.Answer) == "" || !validText(req.Answer, 2000, true) {
		fieldErrors["answer"] = "Answer is required"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	order := 0
	if req.DisplayOrder != nil {
		order = *req.DisplayOrder
	}
	var row map[string]any
	var err error
	if create {
		row, err = s.Store.QueryOne(r.Context(), `INSERT INTO faqs(question,answer,display_order,created_at,updated_at) VALUES($1,$2,$3,NOW(),NOW()) RETURNING id,question,answer,display_order AS "displayOrder"`, req.Question, req.Answer, order)
	} else {
		// Java replaces the whole record: a missing displayOrder resets it to
		// 0 (FaqService.apply) instead of keeping the previous value.
		row, err = s.Store.QueryOne(r.Context(), `UPDATE faqs SET question=$1,answer=$2,display_order=$3,updated_at=NOW() WHERE id=$4 RETURNING id,question,answer,display_order AS "displayOrder"`, req.Question, req.Answer, order, id)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFoundError(w, "Faq", id)
		} else {
			writeError(w, 500, "FAQ could not be saved.")
		}
		return
	}
	status := 200
	if create {
		status = 201
	}
	writeJSON(w, status, row)
}
func (s *Server) deleteFAQ(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	if _, ok := s.execAdmin(w, r, "Faq", id, `DELETE FROM faqs WHERE id=$1`, id); ok {
		writeEmpty(w, 204)
	}
}

type packageRequest struct {
	Title        string      `json:"title"`
	Price        json.Number `json:"price"`
	Features     []string    `json:"features"`
	DisplayOrder *int        `json:"displayOrder"`
	Highlighted  bool        `json:"highlighted"`
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
		err = tx.QueryRow(r.Context(), `INSERT INTO packages(title,price,display_order,highlighted,created_at,updated_at) VALUES($1,$2,$3,$4,NOW(),NOW()) RETURNING id`, title, string(req.Price), order, req.Highlighted).Scan(&pkgID)
	} else {
		// Java replaces the whole record: a missing displayOrder resets it to
		// 0 (PackageItemService.apply) instead of keeping the previous value.
		err = tx.QueryRow(r.Context(), `UPDATE packages SET title=$1,price=$2,display_order=$3,highlighted=$4,updated_at=NOW() WHERE id=$5 RETURNING id`, title, string(req.Price), order, req.Highlighted, id).Scan(&pkgID)
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
	response := map[string]any{"id": pkgID, "title": title, "price": decimalValue(req.Price.String()), "features": features, "displayOrder": order, "highlighted": req.Highlighted}
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
	row, err := s.Store.QueryOne(r.Context(), `SELECT id FROM categories WHERE id=$1`, *req.CategoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Java's PhotoService resolves the category and answers 404.
			notFoundError(w, "Category", *req.CategoryID)
		} else {
			writeError(w, 500, "Photo could not be saved.")
		}
		return
	}
	_ = row
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
	if !create && req.DisplayOrder == nil {
		_ = s.Store.Pool.QueryRow(r.Context(), `SELECT display_order FROM photos WHERE id=$1`, id).Scan(&order)
		result["displayOrder"] = order
	}
	if cat, err := s.Store.QueryOne(r.Context(), `SELECT name FROM categories WHERE id=$1`, *req.CategoryID); err == nil {
		result["categoryName"] = cat["name"]
	}
	imageURL, _ := result["imageUrl"].(string)
	thumbnail, preview := s.photoVariants(imageURL)
	result["thumbnailUrl"], result["previewUrl"] = nullableVariant(thumbnail), nullableVariant(preview)
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

func (s *Server) loadSMTP(r *http.Request) (smtpConfig, error) {
	var c smtpConfig
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT host,port,COALESCE(username,''),COALESCE(encrypted_password,''),encryption,authentication,sender_name,sender_email FROM smtp_settings WHERE id=1`).Scan(&c.Host, &c.Port, &c.Username, &c.EncryptedPassword, &c.Encryption, &c.Authentication, &c.SenderName, &c.SenderEmail)
	return c, err
}
func (s *Server) getSMTPSettings(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadSMTP(r)
	if errors.Is(e, pgx.ErrNoRows) {
		writeJSON(w, 200, map[string]any{"configured": false, "host": "smtp.gmail.com", "port": 587, "username": "", "passwordConfigured": false, "encryption": "STARTTLS", "authentication": true, "senderName": "Elvove", "senderEmail": "elvo.vee@gmail.com"})
		return
	}
	if e != nil {
		writeError(w, 500, "SMTP settings could not be loaded.")
		return
	}
	writeJSON(w, 200, smtpResponse(c))
}
func smtpResponse(c smtpConfig) map[string]any {
	return map[string]any{"configured": true, "host": c.Host, "port": c.Port, "username": c.Username, "passwordConfigured": c.EncryptedPassword != "", "encryption": c.Encryption, "authentication": c.Authentication, "senderName": c.SenderName, "senderEmail": c.SenderEmail}
}
func (s *Server) saveSMTPSettings(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Host           string `json:"host"`
		Port           *int   `json:"port"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		Encryption     string `json:"encryption"`
		Authentication *bool  `json:"authentication"`
		SenderName     string `json:"senderName"`
		SenderEmail    string `json:"senderEmail"`
	}
	if !decodeJSON(w, r, &q) {
		return
	}
	q.Host = strings.TrimSpace(q.Host)
	q.Username = strings.TrimSpace(q.Username)
	q.SenderName = strings.TrimSpace(q.SenderName)
	q.SenderEmail = strings.TrimSpace(q.SenderEmail)
	q.Encryption = strings.ToUpper(strings.TrimSpace(q.Encryption))
	// Mirrors SmtpRequest's bean constraints. Java has no messages on them, so
	// the fieldErrors carry Hibernate Validator's default wording.
	fieldErrors := map[string]string{}
	if q.Host == "" {
		fieldErrors["host"] = "must not be blank"
	}
	if q.Port == nil {
		fieldErrors["port"] = "must not be null"
	} else if *q.Port < 1 {
		fieldErrors["port"] = "must be greater than or equal to 1"
	} else if *q.Port > 65535 {
		fieldErrors["port"] = "must be less than or equal to 65535"
	}
	if q.Encryption == "" {
		fieldErrors["encryption"] = "must not be blank"
	}
	if q.Authentication == nil {
		fieldErrors["authentication"] = "must not be null"
	}
	if q.SenderName == "" {
		fieldErrors["senderName"] = "must not be blank"
	}
	if q.SenderEmail == "" {
		fieldErrors["senderEmail"] = "must not be blank"
	} else if addr, e := mail.ParseAddress(q.SenderEmail); e != nil || addr.Address != q.SenderEmail {
		fieldErrors["senderEmail"] = "must be a well-formed email address"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	if q.Encryption != "NONE" && q.Encryption != "SSL" && q.Encryption != "STARTTLS" {
		// Java throws IllegalArgumentException here, whose text the catch-all
		// handler would turn into a 500; the message is kept, the status is not.
		writeError(w, 400, "Encryption harus NONE, SSL, atau STARTTLS.")
		return
	}
	if !validText(q.SenderEmail, 255, true) || !validText(q.Host, 255, true) || !validText(q.SenderName, 150, true) {
		writeError(w, 400, "SMTP settings are invalid.")
		return
	}
	port := *q.Port
	existing, _ := s.loadSMTP(r)
	encrypted := existing.EncryptedPassword
	var e error
	if strings.TrimSpace(q.Password) != "" {
		if len(s.Config.SMTPEncryptionKey) < 32 {
			writeError(w, 503, "SMTP encryption is not configured.")
			return
		}
		encrypted, e = encryptSMTPPassword(s.Config.SMTPEncryptionKey, q.Password)
		if e != nil {
			writeError(w, 500, "SMTP password could not be encrypted.")
			return
		}
	} else if encrypted == "" && *q.Authentication {
		writeError(w, 400, "Password SMTP wajib diisi untuk autentikasi.")
		return
	}
	_, e = s.Store.Exec(r.Context(), `INSERT INTO smtp_settings(id,host,port,username,encrypted_password,encryption,authentication,sender_name,sender_email) VALUES(1,$1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET host=EXCLUDED.host,port=EXCLUDED.port,username=EXCLUDED.username,encrypted_password=EXCLUDED.encrypted_password,encryption=EXCLUDED.encryption,authentication=EXCLUDED.authentication,sender_name=EXCLUDED.sender_name,sender_email=EXCLUDED.sender_email`, q.Host, q.Port, q.Username, nullableString(encrypted), q.Encryption, *q.Authentication, q.SenderName, q.SenderEmail)
	if e != nil {
		writeError(w, 500, "SMTP settings could not be saved.")
		return
	}
	writeJSON(w, 200, smtpResponse(smtpConfig{q.Host, port, q.Username, encrypted, q.Encryption, *q.Authentication, q.SenderName, q.SenderEmail}))
}
func encryptSMTPPassword(secret []byte, plain string) (string, error) {
	key := sha256.Sum256(secret)
	block, e := aes.NewCipher(key[:])
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(append(nonce, g.Seal(nil, nonce, []byte(plain), nil)...)), nil
}
func decryptSMTPPassword(secret []byte, value string) (string, error) {
	raw, e := base64.StdEncoding.DecodeString(value)
	if e != nil {
		return "", e
	}
	key := sha256.Sum256(secret)
	block, e := aes.NewCipher(key[:])
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	if len(raw) < g.NonceSize() {
		return "", fmt.Errorf("invalid encrypted password")
	}
	plain, e := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	return string(plain), e
}
func (s *Server) smtpDial(r *http.Request, c smtpConfig) (*smtp.Client, error) {
	if c.Authentication && len(s.Config.SMTPEncryptionKey) < 32 {
		return nil, fmt.Errorf("SMTP encryption key is not configured")
	}
	address := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	ctx := context.Background()
	if r != nil && r.Context() != nil {
		ctx = r.Context()
	}
	conn, e := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, "tcp", address)
	if e != nil {
		return nil, e
	}
	deadline := time.Now().Add(8 * time.Second)
	_ = conn.SetDeadline(deadline)
	stopDeadline := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stopDeadline()
	var client *smtp.Client
	if strings.EqualFold(c.Encryption, "SSL") {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
		if e = tlsConn.Handshake(); e != nil {
			conn.Close()
			return nil, e
		}
		client, e = smtp.NewClient(tlsConn, c.Host)
	} else {
		client, e = smtp.NewClient(conn, c.Host)
		if e == nil && strings.EqualFold(c.Encryption, "STARTTLS") {
			ok, _ := client.Extension("STARTTLS")
			if !ok {
				client.Close()
				return nil, fmt.Errorf("SMTP server does not support STARTTLS")
			}
			e = client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
		}
	}
	if e != nil {
		conn.Close()
		return nil, e
	}
	if c.Authentication {
		password, e := decryptSMTPPassword(s.Config.SMTPEncryptionKey, c.EncryptedPassword)
		if e != nil {
			client.Close()
			return nil, fmt.Errorf("SMTP password could not be decrypted")
		}
		auth := smtp.PlainAuth("", c.Username, password, c.Host)
		if e = client.Auth(auth); e != nil {
			client.Close()
			return nil, e
		}
	}
	return client, nil
}
func (s *Server) sendSMTPMessage(r *http.Request, to, subject, htmlBody, textBody string) error {
	req := r
	if req == nil {
		req = &http.Request{}
	}
	c, e := s.loadSMTP(req)
	if e != nil {
		return fmt.Errorf("SMTP belum dikonfigurasi: %w", e)
	}
	client, e := s.smtpDial(req, c)
	if e != nil {
		return fmt.Errorf("SMTP dial: %w", e)
	}
	defer client.Close()
	from := formatMailFrom(c.SenderName, c.SenderEmail)
	if e = client.Mail(c.SenderEmail); e != nil {
		return e
	}
	if e = client.Rcpt(to); e != nil {
		return e
	}
	writer, e := client.Data()
	if e != nil {
		return e
	}
	subjectHeader := mimeWord(subject)
	// Replies go back to the same business inbox the message came from, the
	// same way Java's MimeMessageHelper.setReplyTo does.
	replyTo := ""
	if address := strings.TrimSpace(c.SenderEmail); address != "" {
		replyTo = fmt.Sprintf("Reply-To: %s\r\n", address)
	}
	var message string
	if htmlBody == "" {
		message = fmt.Sprintf("From: %s\r\nTo: %s\r\n%sSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", from, to, replyTo, subjectHeader, base64.StdEncoding.EncodeToString([]byte(textBody)))
	} else {
		boundary := "elvove-alternative-" + newUploadID()
		message = fmt.Sprintf("From: %s\r\nTo: %s\r\n%sSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s\r\n--%s--\r\n", from, to, replyTo, subjectHeader, boundary, boundary, base64.StdEncoding.EncodeToString([]byte(textBody)), boundary, base64.StdEncoding.EncodeToString([]byte(htmlBody)), boundary)
	}
	if _, e = io.WriteString(writer, message); e != nil {
		writer.Close()
		return e
	}
	return writer.Close()
}

// mimeWord encodes a header value as an RFC 2047 encoded-word when it holds
// characters outside US-ASCII, the way JavaMail writes non-ASCII subjects and
// display names.
func mimeWord(value string) string {
	for _, r := range value {
		if r > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(value)) + "?="
		}
	}
	return value
}

// formatMailFrom builds the From header, MIME-encoding a display name that is
// not US-ASCII so the header stays valid for strict servers.
func formatMailFrom(name, address string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return address
	}
	address = strings.TrimSpace(address)
	for _, r := range name {
		if r > 127 {
			return mimeWord(name) + " <" + address + ">"
		}
	}
	from := mail.Address{Name: name, Address: address}
	return from.String()
}

func (s *Server) sendBookingConfirmation(w http.ResponseWriter, r *http.Request) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id < 1 {
		writeError(w, 400, "Invalid resource id.")
		return
	}
	s.confirmationMu.Lock()
	if s.confirming[id] {
		s.confirmationMu.Unlock()
		writeError(w, 400, "Email konfirmasi untuk pemesanan ini sedang dikirim. Tunggu hingga proses selesai.")
		return
	}
	s.confirming[id] = true
	s.confirmationMu.Unlock()
	defer func() { s.confirmationMu.Lock(); delete(s.confirming, id); s.confirmationMu.Unlock() }()
	var booking struct {
		Email         string
		Name          string
		EventDate     string
		EventTime     string
		PackageTitle  string
		PackagePrice  string
		EventLocation *string
	}
	var status string
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT email,name,COALESCE(to_char(event_date,'YYYY-MM-DD'),''),COALESCE(to_char(event_time,'HH24:MI'),''),COALESCE(package_title,''),COALESCE(package_price::text,''),COALESCE(event_location,''),status FROM bookings WHERE id=$1`, id).Scan(&booking.Email, &booking.Name, &booking.EventDate, &booking.EventTime, &booking.PackageTitle, &booking.PackagePrice, &booking.EventLocation, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		// Java's workflow controller uses ResourceNotFoundException with a plain
		// message for this endpoint rather than the generated one.
		writeError(w, 404, "Pemesanan tidak ditemukan.")
		return
	}
	if err != nil {
		writeError(w, 500, "Booking could not be loaded.")
		return
	}
	if status != "ACCEPTED" {
		writeError(w, 400, "Email konfirmasi hanya dapat dikirim untuk pemesanan yang sudah diterima.")
		return
	}
	if strings.TrimSpace(booking.Email) == "" {
		writeError(w, 400, "Pemesanan ini tidak memiliki alamat email klien, sehingga konfirmasi tidak dapat dikirim.")
		return
	}
	subject, htmlBody, textBody := renderBookingConfirmation(bookingEmail{
		Name:          booking.Name,
		EventDate:     booking.EventDate,
		EventTime:     booking.EventTime,
		EventLocation: nullableDeref(booking.EventLocation),
		PackageTitle:  booking.PackageTitle,
		PackagePrice:  booking.PackagePrice,
	}, s.mailSenderName(r), s.mailSenderEmail(r), s.Config.SiteURL)
	if e = s.sendSMTPMessage(r, booking.Email, subject, htmlBody, textBody); e != nil {
		writeError(w, 400, "Email konfirmasi gagal dikirim. Periksa konfigurasi pada Pengaturan Email, lalu coba lagi.")
		return
	}
	var sentAt time.Time
	var count int
	if err = s.Store.Pool.QueryRow(r.Context(), `UPDATE bookings SET confirmation_email_sent_at=NOW(),confirmation_email_count=COALESCE(confirmation_email_count,0)+1 WHERE id=$1 RETURNING confirmation_email_sent_at,confirmation_email_count`, id).Scan(&sentAt, &count); err != nil {
		writeError(w, 400, "Email konfirmasi terkirim, tetapi status pengiriman gagal disimpan. Silakan periksa data pemesanan.")
		return
	}
	writeJSON(w, 200, map[string]any{"message": "Email konfirmasi berhasil dikirim ke " + booking.Email + ".", "sentTo": booking.Email, "confirmationEmailSentAt": wibTimestamp(sentAt), "confirmationEmailCount": count})
}
func nullableDeref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// mailSenderName and mailSenderEmail read the configured sender identity used
// in the From header and in the e-mail templates, mirroring Java's
// SmtpMailService.getSenderName()/getSenderEmail(). They fall back to the
// brand name when SMTP is not configured yet, so a template still renders.
func (s *Server) mailSenderName(r *http.Request) string {
	settings, err := s.loadSMTP(r)
	if err != nil || strings.TrimSpace(settings.SenderName) == "" {
		return emailBrandFallback
	}
	return settings.SenderName
}

func (s *Server) mailSenderEmail(r *http.Request) string {
	settings, err := s.loadSMTP(r)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(settings.SenderEmail)
}
func (s *Server) notifyBookingAdmins(r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var enabled bool
	if err := s.Store.Pool.QueryRow(ctx, `SELECT enabled FROM booking_notification_settings WHERE id=1`).Scan(&enabled); err != nil || !enabled {
		return
	}
	recipients, err := s.Store.QueryRows(ctx, `SELECT email FROM booking_notification_recipients ORDER BY id`)
	if err != nil {
		return
	}
	b := bookingEmail{ID: id}
	if err = s.Store.Pool.QueryRow(ctx, `SELECT name,email,phone,to_char(event_date,'YYYY-MM-DD'),COALESCE(to_char(event_time,'HH24:MI'),''),COALESCE(event_location,''),COALESCE(package_title,''),COALESCE(package_price::text,''),status FROM bookings WHERE id=$1`, id).
		Scan(&b.Name, &b.Email, &b.Phone, &b.EventDate, &b.EventTime, &b.EventLocation, &b.PackageTitle, &b.PackagePrice, &b.Status); err != nil {
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
	for _, recipient := range recipients {
		email, _ := recipient["email"].(string)
		if err := s.sendSMTPMessage(mailReq, email, subject, htmlBody, textBody); err != nil {
			// Java logs the failure class only: no address, no SMTP details.
			log.Printf("Notifikasi booking #%d gagal dikirim ke satu penerima", id)
		}
	}
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
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.Config.MaxRequest)
	if err := r.ParseMultipartForm(s.Config.MaxFileSize); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, 413, uploadTooLargeMessage)
		} else {
			writeError(w, 400, "Upload request is invalid.")
		}
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "Tidak ada berkas foto yang diunggah.")
		return
	}
	defer file.Close()
	path, err := storeUploadedImage(s.Config.UploadDir, s.Config.MaxFileSize, header, file)
	if err != nil {
		status := 400
		message := err.Error()
		if errors.Is(err, errUploadTooLarge) {
			// Spring's multipart parser rejects the part before the controller
			// runs, so its MaxUploadSizeExceededException message is the one
			// Java actually answers with.
			status, message = 413, uploadTooLargeMessage
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, 200, map[string]string{"url": path})
}

const uploadTooLargeMessage = "Ukuran berkas melebihi batas yang diizinkan."

var errUploadTooLarge = errors.New("upload too large")

func storeUploadedImage(root string, max int64, header *multipart.FileHeader, file multipart.File) (string, error) {
	if header.Size < 1 {
		return "", fmt.Errorf("Tidak ada berkas foto yang diunggah.")
	}
	if header.Size > max {
		return "", errUploadTooLarge
	}
	name := filepath.Base(header.Filename)
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return "", fmt.Errorf("Ekstensi berkas foto tidak valid.")
	}
	ext := strings.ToLower(name[dot:])
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("Format foto harus JPG, PNG, atau WebP.")
	}
	tmp, err := os.CreateTemp("", "elvove-image-")
	if err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	limited := io.LimitReader(file, max+1)
	size, err := io.Copy(tmp, limited)
	if err != nil {
		tmp.Close()
		return "", fmt.Errorf("Upload failed.")
	}
	if size > max {
		tmp.Close()
		return "", errUploadTooLarge
	}
	if err = tmp.Close(); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if !validImageSignature(tmpPath, ext) {
		return "", fmt.Errorf("Berkas bukan gambar JPG, PNG, atau WebP yang valid.")
	}
	if err = os.MkdirAll(filepath.Join(root, "originals"), 0750); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if err = os.MkdirAll(filepath.Join(root, "thumbnails"), 0750); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if err = os.MkdirAll(filepath.Join(root, "previews"), 0750); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	id := newUploadID()
	source := filepath.Join(root, "originals", id+ext)
	if err = copyFile(tmpPath, source); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if err = convertWebP(source, filepath.Join(root, "thumbnails", id+".webp"), 800, 55); err != nil {
		removeUploadVariants(root, "/uploads/"+id+ext)
		return "", fmt.Errorf("Foto gagal diproses. Pastikan berkas adalah gambar yang valid.")
	}
	if err = convertWebP(source, filepath.Join(root, "previews", id+".webp"), 1200, 65); err != nil {
		removeUploadVariants(root, "/uploads/"+id+ext)
		_ = os.Remove(source)
		return "", fmt.Errorf("Foto gagal diproses. Pastikan berkas adalah gambar yang valid.")
	}
	return "/uploads/thumbnails/" + id + ".webp", nil
}
func validImageSignature(path, ext string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var header [12]byte
	n, _ := file.Read(header[:])
	jpeg := n >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff
	png := n >= 8 && header[0] == 0x89 && string(header[1:4]) == "PNG" && header[4] == 0x0d && header[5] == 0x0a && header[6] == 0x1a && header[7] == 0x0a
	webp := n >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP"
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg
	case ".png":
		return png
	case ".webp":
		return webp
	}
	return false
}
func newUploadID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
func convertWebP(source, destination string, width, quality int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cwebp", "-quiet", "-resize", strconv.Itoa(width), "0", "-q", strconv.Itoa(quality), source, "-o", destination)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("webp conversion failed: %s", strings.TrimSpace(string(out)))
	}
	info, err := os.Stat(destination)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("webp output missing")
	}
	return nil
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
		Title        string `json:"title"`
		Introduction string `json:"introduction"`
		Content      string `json:"content"`
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
	var updated time.Time
	err := tx.QueryRow(r.Context(), `SELECT id,title,COALESCE(introduction,''),content,version,updated_at FROM site_terms ORDER BY id LIMIT 1 FOR UPDATE`).Scan(&id, &title, &introduction, &content, &version, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		version = "1"
		err = tx.QueryRow(r.Context(), `INSERT INTO site_terms(title,introduction,content,version,created_at,updated_at) VALUES($1,$2,$3,$4,NOW(),NOW()) RETURNING id,title,introduction,content,version,updated_at`, req.Title, req.Introduction, req.Content, version).Scan(&id, &title, &introduction, &content, &version, &updated)
	} else if err == nil {
		if title != req.Title || nullableDeref(&introduction) != req.Introduction || content != req.Content {
			n, _ := strconv.Atoi(version)
			version = strconv.Itoa(n + 1)
		}
		err = tx.QueryRow(r.Context(), `UPDATE site_terms SET title=$1,introduction=$2,content=$3,version=$4,updated_at=NOW() WHERE id=$5 RETURNING id,title,introduction,content,version,updated_at`, req.Title, req.Introduction, req.Content, version, id).Scan(&id, &title, &introduction, &content, &version, &updated)
	}
	if err != nil {
		writeError(w, 500, "Terms could not be saved.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "Terms could not be saved.")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "title": title, "introduction": introduction, "content": content, "version": version, "updatedAt": updated})
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
	allowed := map[string][]string{"PENDING": {"ACCEPTED", "RESCHEDULE_REQUESTED"}, "ACCEPTED": {"COMPLETED", "RESCHEDULE_REQUESTED"}, "RESCHEDULE_REQUESTED": {"ACCEPTED", "COMPLETED"}}
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
	if _, err := s.Store.Exec(r.Context(), `UPDATE bookings SET status=$1,updated_at=NOW() WHERE id=$2`, target, id); err != nil {
		writeError(w, 500, "Booking could not be updated.")
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
	rows, err := s.Store.QueryRows(r.Context(), bookingColumns+` WHERE id=$1`, id)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("booking %d could not be loaded", id)
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
	if current != "RESCHEDULE_REQUESTED" {
		// Java allows this only from PENDING or ACCEPTED; anything else reports
		// the same transition message as validateTransition.
		valid := current == "PENDING" || current == "ACCEPTED"
		if !valid {
			writeError(w, 400, fmt.Sprintf(`Status pemesanan tidak dapat diubah dari "%s" menjadi "%s".`, statusLabel(current), statusLabel("RESCHEDULE_REQUESTED")))
			return
		}
	}
	var newStatus string
	if err := tx.QueryRow(r.Context(), `UPDATE bookings SET event_date=$1,event_time=$2,status='RESCHEDULE_REQUESTED',updated_at=NOW() WHERE id=$3 RETURNING status`, date, clock.Format("15:04:05"), id).Scan(&newStatus); err != nil {
		writeError(w, 500, "Booking could not be rescheduled.")
		return
	}
	subject, htmlBody, textBody := renderBookingReschedule(b, s.mailSenderName(r), s.mailSenderEmail(r), s.Config.SiteURL)
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
	if _, ok := s.execAdmin(w, r, "Booking", id, `DELETE FROM bookings WHERE id=$1`, id); ok {
		writeEmpty(w, 204)
	}
}

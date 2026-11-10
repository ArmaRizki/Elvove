package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/elvove/photography/backend-go/internal/config"
	"github.com/elvove/photography/backend-go/internal/store"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	Config         config.Config
	Store          *store.Store
	Auth           *Auth
	Limiter        *RateLimiter
	confirmationMu sync.Mutex
	confirming     map[int64]bool
}

type decimalValue string

var decimalPattern = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)
var phonePattern = regexp.MustCompile(`^[0-9+\-\s]{7,15}$`)

func (d decimalValue) MarshalJSON() ([]byte, error) {
	if !decimalPattern.MatchString(string(d)) {
		return nil, fmt.Errorf("invalid database decimal")
	}
	return []byte(d), nil
}

func NewServer(cfg config.Config, db *store.Store) *Server {
	limiter := NewRateLimiter()
	s := &Server{Config: cfg, Store: db, Limiter: limiter, confirming: make(map[int64]bool)}
	s.Auth = &Auth{Store: db, Secret: cfg.JWTSecret, Limiter: limiter, MaxAge: 24 * time.Hour}
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.Store.Pool.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "Database unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/auth/login", s.Auth.Login)
	mux.HandleFunc("GET /api/public/about", s.getAbout)
	mux.HandleFunc("GET /api/public/contact", s.getContact)
	mux.HandleFunc("GET /api/public/terms", s.getTerms)
	mux.HandleFunc("GET /api/public/social-links", s.getSocialLinks)
	mux.HandleFunc("GET /api/public/categories", s.getCategories)
	mux.HandleFunc("GET /api/public/packages", s.getPackages)
	mux.HandleFunc("GET /api/public/photos", s.getPhotos)
	mux.Handle("/uploads/originals/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Java's SecurityConfig has .requestMatchers("/uploads/originals/**").denyAll(),
		// and its HttpStatusEntryPoint turns the denial into an empty 401, which
		// production confirms (403 would leak that the path exists).
		writeEmpty(w, http.StatusUnauthorized)
	}))
	mux.Handle("/uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/uploads/")
		decoded, decodeErr := url.PathUnescape(name)
		clean := filepath.Clean(decoded)
		if decodeErr != nil || clean == "originals" || strings.HasPrefix(clean, "originals"+string(filepath.Separator)) || filepath.IsAbs(decoded) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			writeNotFound(w)
			return
		}
		root, err := filepath.Abs(s.Config.UploadDir)
		resolved, resolveErr := filepath.Abs(filepath.Join(root, clean))
		if err != nil || resolveErr != nil || (resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator))) {
			writeNotFound(w)
			return
		}
		if info, err := os.Stat(resolved); err != nil || !info.Mode().IsRegular() {
			writeNotFound(w)
			return
		}
		http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.Config.UploadDir))).ServeHTTP(w, r)
	}))

	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/admin/dashboard/summary", s.dashboardSummary)
	admin.HandleFunc("GET /api/admin/bookings", s.getBookings)
	admin.HandleFunc("PATCH /api/admin/bookings/{id}/status", s.updateBookingStatus)
	admin.HandleFunc("POST /api/admin/bookings/{id}/accept", s.acceptBooking)
	admin.HandleFunc("POST /api/admin/bookings/{id}/confirmation-email", s.sendBookingConfirmation)
	admin.HandleFunc("PATCH /api/admin/bookings/{id}/reschedule", s.rescheduleBooking)
	admin.HandleFunc("DELETE /api/admin/bookings/{id}", s.deleteBooking)
	admin.HandleFunc("POST /api/admin/upload", s.upload)
	admin.HandleFunc("POST /api/admin/photos", s.createPhoto)
	admin.HandleFunc("PUT /api/admin/photos/{id}", s.updatePhoto)
	admin.HandleFunc("DELETE /api/admin/photos/{id}", s.deletePhoto)
	admin.HandleFunc("POST /api/admin/categories", s.createCategory)
	admin.HandleFunc("PUT /api/admin/categories/{id}", s.updateCategory)
	admin.HandleFunc("DELETE /api/admin/categories/{id}", s.deleteCategory)
	admin.HandleFunc("POST /api/admin/packages", s.createPackage)
	admin.HandleFunc("PUT /api/admin/packages/{id}", s.updatePackage)
	admin.HandleFunc("DELETE /api/admin/packages/{id}", s.deletePackage)
	admin.HandleFunc("PUT /api/admin/about", s.updateAbout)
	admin.HandleFunc("PUT /api/admin/contact", s.updateContact)
	admin.HandleFunc("PUT /api/admin/terms", s.updateTerms)
	admin.HandleFunc("PUT /api/admin/social-links", s.updateSocialLinks)
	admin.HandleFunc("GET /api/admin/settings/smtp", s.getSMTPSettings)
	admin.HandleFunc("PUT /api/admin/settings/smtp", s.saveSMTPSettings)
	admin.HandleFunc("GET /api/admin/settings/booking-notifications", s.getNotificationSettings)
	admin.HandleFunc("PUT /api/admin/settings/booking-notifications", s.saveNotificationSettings)
	admin.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeNotFound(w) })
	mux.Handle("/api/admin/", s.Auth.Middleware(s.Auth.RequireAdmin(admin)))
	mux.HandleFunc("POST /api/bookings", s.createBooking)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Spring Security runs before the router, so an unknown path only reaches
		// NoResourceFoundException (404 + ApiError) when its matcher permits it.
		// Everything else is rejected by anyRequest().authenticated() with an empty
		// 401, which is what production answers on / and /api/<unknown>.
		permitted := strings.HasPrefix(r.URL.Path, "/api/auth/") ||
			(r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/public/")) ||
			(r.Method == http.MethodPost && r.URL.Path == "/api/bookings") ||
			strings.HasPrefix(r.URL.Path, "/uploads/")
		if permitted {
			writeNotFound(w)
			return
		}
		writeEmpty(w, http.StatusUnauthorized)
	})

	return s.cors(s.recover(mux))
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(s.Config.Origins))
	for _, origin := range s.Config.Origins {
		allowed[origin] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders(r.Header.Get("Access-Control-Request-Headers")))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "3600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allowHeaders mirrors Java's CorsConfiguration.setAllowedHeaders("*"): the
// requested headers are echoed back, so a custom header a client sends is not
// blocked. Browsers always ask for at least one header on a preflight.
func allowHeaders(requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "Authorization, Content-Type"
	}
	return requested
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeError(w, http.StatusInternalServerError, "Internal server error.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getAbout(w http.ResponseWriter, r *http.Request) {
	row, err := s.Store.QueryOne(r.Context(), `SELECT id, introduction, years_of_experience AS "yearsOfExperience", photography_style AS "photographyStyle", achievements, story, image_url AS "imageUrl" FROM about_info ORDER BY id LIMIT 1`)
	s.rowOrError(w, row, err, map[string]any{"id": nil, "introduction": nil, "yearsOfExperience": nil, "photographyStyle": nil, "achievements": nil, "story": nil, "imageUrl": nil})
}

func (s *Server) getContact(w http.ResponseWriter, r *http.Request) {
	row, err := s.Store.QueryOne(r.Context(), `SELECT id, phone, email, address, google_map_link AS "googleMapLink", whatsapp_number AS "whatsappNumber" FROM contact_info ORDER BY id LIMIT 1`)
	s.rowOrError(w, row, err, map[string]any{"id": nil, "phone": nil, "email": nil, "address": nil, "googleMapLink": nil, "whatsappNumber": nil})
}

func (s *Server) getTerms(w http.ResponseWriter, r *http.Request) {
	row, err := s.Store.QueryOne(r.Context(), `SELECT id, title, introduction, content, version, updated_at AS "updatedAt" FROM site_terms ORDER BY id LIMIT 1`)
	if errors.Is(err, pgx.ErrNoRows) {
		// Java's TermsService.get() persists the default row, so the first read
		// answers with an id and a timestamp like every later one.
		row, err = s.Store.QueryOne(r.Context(), `INSERT INTO site_terms(title,introduction,content,version,created_at,updated_at) VALUES($1,$2,$3,'1',$4,$4) RETURNING id,title,introduction,content,version,updated_at AS "updatedAt"`, "Terms & Conditions", defaultTermsIntroduction, defaultTermsContent, nowWallClock())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat data.")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

// defaultTermsIntroduction is the default introduction Java's TermsService seeds.
const defaultTermsIntroduction = "Please read these terms carefully before submitting your photography booking inquiry."

func (s *Server) getSocialLinks(w http.ResponseWriter, r *http.Request) {
	row, err := s.Store.QueryOne(r.Context(), `SELECT id, instagram, threads, tiktok, facebook, youtube, pinterest, linkedin FROM social_links ORDER BY id LIMIT 1`)
	s.rowOrError(w, row, err, map[string]any{"id": nil, "instagram": nil, "threads": nil, "tiktok": nil, "facebook": nil, "youtube": nil, "pinterest": nil, "linkedin": nil})
}

func (s *Server) getCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.QueryRows(r.Context(), `SELECT id, name, slug FROM categories`)
	s.rowsOrError(w, rows, err)
}

func (s *Server) getPackages(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT p.id, p.title, p.price, p.display_order, p.highlighted, COALESCE((SELECT json_agg(pf.feature) FROM package_features pf WHERE pf.package_id = p.id), '[]'::json) AS features FROM packages p ORDER BY p.display_order ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat paket.")
		return
	}
	defer rows.Close()
	type item struct {
		ID           int64           `json:"id"`
		Title        string          `json:"title"`
		Price        decimalValue    `json:"price"`
		DisplayOrder int             `json:"displayOrder"`
		Highlighted  bool            `json:"highlighted"`
		Features     json.RawMessage `json:"features"`
	}
	items := make([]item, 0)
	for rows.Next() {
		var current item
		if err := rows.Scan(&current.ID, &current.Title, &current.Price, &current.DisplayOrder, &current.Highlighted, &current.Features); err != nil {
			writeError(w, http.StatusInternalServerError, "Gagal memuat paket.")
			return
		}
		items = append(items, current)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat paket.")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getPhotos(w http.ResponseWriter, r *http.Request) {
	query := `SELECT p.id, p.title, p.description, p.image_url AS "imageUrl", p.featured, p.display_order AS "displayOrder", c.id AS "categoryId", c.name AS "categoryName" FROM photos p LEFT JOIN categories c ON c.id = p.category_id`
	args := []any{}
	switch {
	case r.URL.Query().Get("categoryId") != "":
		// A non-numeric value fails Spring's Long conversion, which the catch-all
		// reports as an unreadable body; 0 and below simply match no photo.
		id, err := strconv.ParseInt(r.URL.Query().Get("categoryId"), 10, 64)
		if err != nil {
			writeUnreadableBody(w)
			return
		}
		query += ` WHERE c.id = $1 ORDER BY p.display_order ASC`
		args = append(args, id)
	case r.URL.Query().Get("featured") == "true":
		query += ` WHERE p.featured = true ORDER BY p.display_order ASC`
	case r.URL.Query().Get("latest") == "true":
		query = `SELECT p.id, p.title, p.description, p.image_url AS "imageUrl", p.featured, p.display_order AS "displayOrder", c.id AS "categoryId", c.name AS "categoryName" FROM (SELECT * FROM photos ORDER BY created_at DESC LIMIT 12) p LEFT JOIN categories c ON c.id = p.category_id ORDER BY p.created_at DESC`
	default:
		query += ` ORDER BY p.display_order ASC`
	}
	rows, err := s.Store.QueryRows(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat foto.")
		return
	}
	for _, row := range rows {
		imageURL, _ := row["imageUrl"].(string)
		thumbnail, preview := s.photoVariants(imageURL)
		row["thumbnailUrl"], row["previewUrl"] = nullableVariant(thumbnail), nullableVariant(preview)
	}
	writeJSON(w, http.StatusOK, rows)
}

// uploadFilename mirrors Java's FileStorageService.safeFilename: only a plain
// file name (no sub-directory, no traversal) below /uploads/ is accepted.
var uploadFilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func uploadFilename(imageURL string) string {
	if !strings.HasPrefix(imageURL, "/uploads/") {
		return ""
	}
	filename := imageURL[strings.LastIndex(imageURL, "/")+1:]
	if !uploadFilenamePattern.MatchString(filename) || !strings.Contains(filename, ".") {
		return ""
	}
	return filename
}

// getThumbnailURL mirrors FileStorageService.getThumbnailUrl: a photo that
// predates the WebP variants gets them generated on first read, so a legacy row
// never answers 404 for its thumbnail. It falls back to the original URL when
// the conversion fails, exactly like Java.
func (s *Server) getThumbnailURL(imageURL string) string {
	if strings.TrimSpace(imageURL) == "" {
		return ""
	}
	if strings.HasPrefix(imageURL, "/uploads/thumbnails/") {
		return imageURL
	}
	filename := uploadFilename(imageURL)
	if filename == "" {
		return ""
	}
	id := strings.TrimSuffix(filename, filepath.Ext(filename))
	root := s.Config.UploadDir
	thumbnail := filepath.Join(root, "thumbnails", id+".webp")
	if _, err := os.Stat(thumbnail); err == nil {
		return "/uploads/thumbnails/" + id + ".webp"
	}
	legacy := filepath.Join(root, filename)
	// The name has no separator, but the check keeps a crafted row from reading
	// outside the upload directory.
	if !strings.HasPrefix(legacy, filepath.Clean(root)+string(filepath.Separator)) {
		return ""
	}
	if info, err := os.Stat(legacy); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	preview := filepath.Join(root, "previews", id+".webp")
	if err := os.MkdirAll(filepath.Dir(thumbnail), 0750); err != nil {
		return imageURL
	}
	if err := os.MkdirAll(filepath.Dir(preview), 0750); err != nil {
		return imageURL
	}
	if err := convertWebP(legacy, thumbnail, 800, 55); err != nil {
		log.Printf("failed to generate legacy thumbnail for %s: %v", imageURL, err)
		return imageURL
	}
	if err := convertWebP(legacy, preview, 1200, 65); err != nil {
		log.Printf("failed to generate legacy preview for %s: %v", imageURL, err)
		return imageURL
	}
	return "/uploads/thumbnails/" + id + ".webp"
}

// nullableVariant answers JSON null when Java's getThumbnailUrl/getPreviewUrl
// would return null (a photo whose imageUrl is not an upload), instead of "".
func nullableVariant(url string) any {
	if url == "" {
		return nil
	}
	return url
}

func (s *Server) photoVariants(imageURL string) (string, string) {
	thumbnail := s.getThumbnailURL(imageURL)
	if !strings.HasPrefix(thumbnail, "/uploads/thumbnails/") {
		return thumbnail, ""
	}
	return thumbnail, "/uploads/previews/" + strings.TrimPrefix(thumbnail, "/uploads/thumbnails/")
}

func (s *Server) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT (SELECT count(*) FROM photos), (SELECT count(*) FROM bookings), (SELECT count(*) FROM packages)`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat ringkasan dasbor.")
		return
	}
	defer rows.Close()
	var photos, bookings, packages int64
	if !rows.Next() || rows.Scan(&photos, &bookings, &packages) != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat ringkasan dasbor.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"totalPhotos": photos, "totalBookings": bookings, "totalPackages": packages})
}

// bookingColumns keeps the list and single-record responses in one shape; both
// paths format price and time the same way (Java returns one BookingResponse).
// Spring Boot disables WRITE_DATES_AS_TIMESTAMPS, so Jackson writes LocalTime as
// the ISO string "09:30:00", which is exactly what the admin UI parses.
const bookingColumns = `SELECT id, name, email, phone, event_date AS "eventDate", to_char(event_time, 'HH24:MI:SS') AS "eventTime", package_title AS "packageTitle", package_price::text AS "packagePrice", event_location AS "eventLocation", budget, message, status, created_at AS "createdAt", updated_at AS "updatedAt", terms_accepted AS "termsAccepted", terms_accepted_at AS "termsAcceptedAt", terms_version AS "termsVersion", confirmation_email_sent_at AS "confirmationEmailSentAt", COALESCE(confirmation_email_count, 0) AS "confirmationEmailCount" FROM bookings`

func normalizeBooking(booking map[string]any) {
	if price, ok := booking["packagePrice"].(string); ok && decimalPattern.MatchString(price) {
		booking["packagePrice"] = decimalValue(price)
	}
}

func (s *Server) getBookings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.QueryRows(r.Context(), bookingColumns+` ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat pemesanan.")
		return
	}
	for _, booking := range rows {
		normalizeBooking(booking)
	}
	s.rowsOrError(w, rows, nil)
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name          string  `json:"name"`
		Email         string  `json:"email"`
		Phone         string  `json:"phone"`
		EventDate     string  `json:"eventDate"`
		EventTime     string  `json:"eventTime"`
		PackageID     *int64  `json:"packageId"`
		EventLocation *string `json:"eventLocation"`
		Budget        *string `json:"budget"`
		Message       *string `json:"message"`
		TermsAccepted *bool   `json:"termsAccepted"`
		TermsVersion  string  `json:"termsVersion"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := decoder.Decode(&request); err != nil {
		writeUnreadableBody(w)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeUnreadableBody(w)
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(request.Email)
	request.Phone = strings.TrimSpace(request.Phone)
	// Mirrors BookingRequest's bean validation: Java answers 400 with fieldErrors
	// naming each rejected field, so the public form can highlight them.
	fieldErrors := map[string]string{}
	if request.Name == "" {
		fieldErrors["name"] = "Name is required"
	} else if utf8.RuneCountInString(request.Name) > 255 {
		fieldErrors["name"] = "Name is too long"
	}
	if request.Email == "" {
		fieldErrors["email"] = "Email is required"
	} else if address, emailErr := mail.ParseAddress(request.Email); emailErr != nil || address.Address != request.Email || utf8.RuneCountInString(request.Email) > 255 {
		fieldErrors["email"] = "Enter a valid email address"
	}
	if request.Phone == "" {
		fieldErrors["phone"] = "Phone is required"
	} else if !phonePattern.MatchString(request.Phone) {
		fieldErrors["phone"] = "Enter a valid phone number"
	}
	eventDate, _ := time.Parse("2006-01-02", request.EventDate)
	if message := eventDateError(request.EventDate); message != "" {
		fieldErrors["eventDate"] = message
	}
	eventTime, timeErr := time.Parse("15:04", request.EventTime)
	if timeErr != nil {
		eventTime, timeErr = time.Parse("15:04:05", request.EventTime)
	}
	if request.EventTime == "" {
		fieldErrors["eventTime"] = "Event time is required"
	} else if timeErr != nil {
		fieldErrors["eventTime"] = "Event time must be a valid time"
	}
	// @NotNull only: a present but non-positive id passes validation and then
	// reaches the package lookup, which answers 404 — matching Java, where
	// packageId 0 or -5 is "PackageItem not found with id: 0" rather than a 400.
	if request.PackageID == nil {
		fieldErrors["packageId"] = "Package is required"
	}
	if request.TermsAccepted == nil {
		fieldErrors["termsAccepted"] = "Terms acceptance is required"
	} else if !*request.TermsAccepted {
		fieldErrors["termsAccepted"] = "You must accept the Terms & Conditions"
	}
	if strings.TrimSpace(request.TermsVersion) == "" {
		fieldErrors["termsVersion"] = "Terms version is required"
	}
	// Java has no length constraint on these; the database does, so a too-long
	// value is reported here rather than as an opaque 500.
	if request.Message != nil && utf8.RuneCountInString(*request.Message) > 2000 {
		fieldErrors["message"] = "Message is too long"
	}
	if request.EventLocation != nil && utf8.RuneCountInString(*request.EventLocation) > 255 {
		fieldErrors["eventLocation"] = "Event location is too long"
	}
	if request.Budget != nil && utf8.RuneCountInString(*request.Budget) > 255 {
		fieldErrors["budget"] = "Budget is too long"
	}
	if utf8.RuneCountInString(request.TermsVersion) > 20 {
		fieldErrors["termsVersion"] = "Terms version is too long"
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	// Java runs @Valid first and only then consumes the rate-limit slot, so an
	// invalid request must not count against the booking quota for this IP.
	if !s.Limiter.Allow("booking:"+ClientIP(r), 5, 10*time.Minute) {
		writeError(w, http.StatusTooManyRequests, "Too many booking requests. Please try again in a few minutes.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	// fail logs the real cause and answers the Java-shaped 500. Without the log a
	// scan error (a NULL column, for instance) is indistinguishable from a
	// database outage from the outside, which is how this path failed unnoticed.
	fail := func(step string, cause error) {
		log.Printf("create booking %s: %v", step, cause)
		writeError(w, http.StatusInternalServerError, "Pemesanan gagal disimpan.")
	}
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		fail("begin", err)
		return
	}
	defer tx.Rollback(ctx)
	// Validation guarantees a non-nil id by this point.
	packageID := *request.PackageID
	var packageTitle, packagePrice string
	err = tx.QueryRow(ctx, `SELECT title, price::text FROM packages WHERE id = $1`, packageID).Scan(&packageTitle, &packagePrice)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Java raises ResourceNotFoundException -> 404 with the id echoed back.
			notFoundError(w, "PackageItem", packageID)
		} else {
			fail("load package", err)
		}
		return
	}
	var termsVersion, termsTitle, termsIntroduction, termsContent string
	// introduction is nullable in the schema and production has it NULL, so a
	// bare scan into string fails with "cannot scan NULL into *string" and turns
	// every booking into a 500. Java's entity maps the column to a String, which
	// reads NULL as null, so COALESCE keeps the Go port equally tolerant.
	err = tx.QueryRow(ctx, `SELECT version, title, COALESCE(introduction,''), COALESCE(content,'') FROM site_terms ORDER BY id LIMIT 1`).Scan(&termsVersion, &termsTitle, &termsIntroduction, &termsContent)
	if errors.Is(err, pgx.ErrNoRows) {
		termsVersion = "1"
		termsTitle = "Terms & Conditions"
		termsIntroduction = defaultTermsIntroduction
		termsContent = defaultTermsContent
		err = tx.QueryRow(ctx, `INSERT INTO site_terms (title, introduction, content, version, created_at, updated_at) VALUES ($1,$2,$3,'1',$4,$4) RETURNING version`, termsTitle, termsIntroduction, termsContent, nowWallClock()).Scan(&termsVersion)
		if err != nil {
			fail("seed terms", err)
			return
		}
	} else if err != nil {
		fail("load terms", err)
		return
	}
	if termsVersion != request.TermsVersion {
		writeError(w, http.StatusConflict, "Terms & Conditions have changed. Please review the latest version.")
		return
	}
	now := nowWallClock()
	var id int64
	var createdAt, updatedAt time.Time
	var confirmationCount int
	// Java's BookingRequest takes eventTime as LocalTime; Spring writes it back as
	// the ISO string "09:30:00".
	eventTimeString := eventTime.Format("15:04:05")
	err = tx.QueryRow(ctx, `INSERT INTO bookings (name, email, phone, event_date, event_time, package_title, package_price, event_location, budget, message, terms_accepted, terms_accepted_at, terms_version, status, confirmation_email_count, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,TRUE,$11,$12,'PENDING',0,$11,$11) RETURNING id, created_at, updated_at, confirmation_email_count`, request.Name, request.Email, request.Phone, eventDate, eventTimeString, packageTitle, packagePrice, optionalText(request.EventLocation), optionalText(request.Budget), optionalText(request.Message), now, request.TermsVersion).Scan(&id, &createdAt, &updatedAt, &confirmationCount)
	if err != nil {
		fail("insert booking", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		fail("commit", err)
		return
	}
	go s.notifyBookingAdmins(r, id)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "name": request.Name, "email": request.Email, "phone": request.Phone,
		"eventDate": eventDate.Format("2006-01-02"),
		"eventTime": eventTimeString, "packageTitle": packageTitle,
		"packagePrice": decimalValue(packagePrice), "eventLocation": optionalText(request.EventLocation),
		"budget": optionalText(request.Budget), "message": optionalText(request.Message),
		"status": "PENDING", "createdAt": wibTimestamp(createdAt), "updatedAt": wibTimestamp(updatedAt),
		"termsAccepted": true, "termsAcceptedAt": wibTimestamp(now), "termsVersion": request.TermsVersion,
		"confirmationEmailSentAt": nil, "confirmationEmailCount": confirmationCount,
	})
}

const defaultTermsContent = `# Booking & Reservation

- All bookings must be submitted through the official ELVOVE booking form.
- A booking is confirmed only after ELVOVE has approved the requested schedule and received the required down payment (DP).
- Submitting a booking inquiry does not automatically guarantee availability.

# Payment & Down Payment

- A down payment is required to secure a confirmed photography session.
- The down payment is non-refundable if the client cancels the session unilaterally.
- The remaining balance must be paid by the due date stated in the invoice.
- Additional services or charges outside the selected package will be communicated to the client.

# Rescheduling & Cancellation

- Rescheduling requests must be submitted at least three (3) days before the scheduled session.
- All rescheduling requests are subject to photographer availability.
- Requests made less than three (3) days before the session may be declined or subject to additional arrangements.
- If ELVOVE is unable to fulfill a confirmed booking, an alternative schedule or appropriate resolution will be discussed with the client.

# Photography Session

- Clients are expected to arrive on time and be ready at the agreed session location.
- Late arrival may reduce the available photography duration without extending the original session end time.
- Additional session time is subject to photographer availability and may incur overtime charges.
- ELVOVE will make reasonable creative and technical decisions during the session to achieve the intended photographic results.

# Package & Photo Editing

- Photography sessions may include unlimited captured shots, but the number of edited photographs is determined by the selected package.
- Clients may select photographs for editing according to the quantity included in their package.
- Additional edited photographs or special editing requests may incur extra charges.
- Final editing follows ELVOVE's photographic style and agreed package specifications.

# Photo Delivery

- Final photographs will be delivered digitally through a cloud storage link or another agreed delivery method.
- Delivery timelines will be communicated according to the selected package and project requirements.
- Clients are responsible for downloading and safely backing up their delivered photographs.
- The availability period of the download link will be communicated by ELVOVE.

# Client Responsibilities & Additional Costs

- Makeup artists, outfits, accessories, and personal props are the client's responsibility unless explicitly included in the package.
- Location entrance fees, photography permits, parking fees, and other location-related charges are the client's responsibility unless otherwise agreed.
- Clients are responsible for obtaining any required permissions for their chosen photography location.
- Clients should take reasonable care of their personal belongings during the session.

# Weather & Force Majeure

- If severe weather or circumstances beyond reasonable control make the session unsafe or impractical, ELVOVE and the client will discuss an alternative schedule.
- Weather-related rescheduling agreed upon by both parties will not incur an additional rescheduling fee.
- Alternative dates remain subject to photographer availability.

# Copyright & Portfolio Usage

- Copyright and usage rights are governed by applicable law and any separate written agreement between ELVOVE and the client.
- ELVOVE may use selected photographs for portfolios, websites, social media, and promotional materials.
- Clients who do not wish their photographs to be published must notify ELVOVE before the photography session.
- ELVOVE will respect any publication restrictions expressly agreed upon with the client.

# Agreement

- By submitting a booking inquiry and accepting these Terms & Conditions, the client acknowledges having read and understood the applicable booking policies.
- A confirmed booking remains subject to the agreed package, invoice, and any additional written arrangements.
- Any changes to the terms applicable to a confirmed booking should be communicated and agreed upon by the relevant parties.
`

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// optionalText keeps the distinction Jackson makes for an optional String:
// a field absent from the JSON stays NULL, while "" is stored as "". Java's
// entity setters store the value they are given, so an empty field the UI sends
// must not turn into NULL here.
func optionalText(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func (s *Server) rowOrError(w http.ResponseWriter, row map[string]any, err error, empty map[string]any) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, empty)
			return
		}
		writeError(w, http.StatusInternalServerError, "Gagal memuat data.")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) rowsOrError(w http.ResponseWriter, rows []map[string]any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memuat data.")
		return
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) notImplemented(feature string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, fmt.Sprintf("%s is not implemented in the Go migration yet.", feature))
	}
}

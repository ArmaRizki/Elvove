package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elvove/photography/backend-go/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestMiddlewareAcceptsValidHS256Token(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	auth := &Auth{Secret: secret, Store: stubAdminStore{found: true}}
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := authenticatedUsername(r.Context()); got != "admin" {
			t.Fatalf("username in context = %q, want admin", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	auth.Middleware(auth.RequireAdmin(next)).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMiddlewareAcceptsJavaHS512Token(t *testing.T) {
	// JJWT (Java) picks HS512 when JWT_SECRET is longer than 48 bytes, so a
	// token the Java backend issued must still verify here.
	secret := []byte(strings.Repeat("s", 64))
	auth := &Auth{Secret: secret, Store: stubAdminStore{found: true}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	auth.Middleware(auth.RequireAdmin(next)).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestJWTClaimsMatchJavaSubjectShape(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return secret, nil })
	if err != nil {
		t.Fatal(err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "admin" || claims["username"] != nil {
		t.Fatalf("JWT claims do not match Java subject format: %#v", claims)
	}
}

func TestPhotoVariants(t *testing.T) {
	server := &Server{Config: config.Config{UploadDir: t.TempDir()}}
	tests := []struct{ input, thumbnail, preview string }{
		{"/uploads/thumbnails/a.webp", "/uploads/thumbnails/a.webp", "/uploads/previews/a.webp"},
		// Legacy row whose original is gone: Java answers null, not a dead URL.
		{"/uploads/old-photo.jpg", "", ""},
		// A remote or malformed URL is not an upload; Java's safeFilename says no.
		{"https://example.com/a.jpg", "", ""},
		{"/uploads/../secret.jpg", "", ""},
		{"", "", ""},
	}
	for _, test := range tests {
		thumbnail, preview := server.photoVariants(test.input)
		if thumbnail != test.thumbnail || preview != test.preview {
			t.Errorf("photoVariants(%q) = %q, %q; want %q, %q", test.input, thumbnail, preview, test.thumbnail, test.preview)
		}
	}
}

func TestLegacyPhotoGeneratesVariantsOnRead(t *testing.T) {
	root := t.TempDir()
	// The call is: cwebp -quiet -resize <w> 0 -q <q> <source> -o <destination>.
	bin := t.TempDir()
	script := "#!/bin/sh\ncat \"$7\" > \"$9\"\n"
	if err := os.WriteFile(bin+"/cwebp", []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	if err := os.WriteFile(root+"/legacy.jpg", []byte("not-a-real-jpeg"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := &Server{Config: config.Config{UploadDir: root}}
	thumbnail, preview := server.photoVariants("/uploads/legacy.jpg")
	if thumbnail != "/uploads/thumbnails/legacy.webp" || preview != "/uploads/previews/legacy.webp" {
		t.Fatalf("photoVariants = %q, %q; want generated variant URLs", thumbnail, preview)
	}
	for _, path := range []string{root + "/thumbnails/legacy.webp", root + "/previews/legacy.webp"} {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Fatalf("variant %s was not written: %v", path, err)
		}
	}
}

func TestOriginalUploadsAreNotServed(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root+"/originals", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root+"/originals/private.jpg", []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := &Server{Config: config.Config{UploadDir: root}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/uploads/originals/private.jpg", nil)
	server.Handler().ServeHTTP(recorder, request)
	// Java's denyAll() is answered by its HttpStatusEntryPoint, so production
	// returns 401 (checked against the live Java container), not 403.
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("original file status = %d, want 401 like Java's denyAll()", recorder.Code)
	}
}

func TestDecimalJSONRemainsNumber(t *testing.T) {
	encoded, err := json.Marshal(decimalValue("1250.50"))
	if err != nil || string(encoded) != "1250.50" {
		t.Fatalf("decimal marshal = %s, %v", encoded, err)
	}
}

func TestMiddlewareRejectsExpiredToken(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	auth := &Auth{Secret: secret}
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not receive an expired token")
	})).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestRequireAdminRejectsMissingIdentity(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil)
	request = request.WithContext(context.Background())
	NewServer(config.Config{}, nil).Auth.RequireAdmin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not receive a request without admin identity")
	})).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

// stubAdminStore answers the membership lookup RequireAdmin performs, mirroring
// Java's AdminUserDetailsService reload on every admin request.
type stubAdminStore struct{ found bool }

func (s stubAdminStore) QueryOne(context.Context, string, ...any) (map[string]any, error) {
	return map[string]any{"found": s.found}, nil
}

func TestRequireAdminRejectsUnknownSubject(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	auth := &Auth{Secret: secret, Store: stubAdminStore{found: false}}
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "removed-admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/bookings", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	auth.Middleware(auth.RequireAdmin(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not receive a subject that is not an admin")
	}))).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

// credentialStore answers the admins lookup; only the exact username matches,
// the way Java's AdminUserDetailsService.findByUsername works.
type credentialStore struct{ username, hash string }

func (s credentialStore) QueryOne(_ context.Context, _ string, args ...any) (map[string]any, error) {
	if value, _ := args[0].(string); value == s.username {
		return map[string]any{"username": s.username, "email": s.username + "@example.com", "password": s.hash}, nil
	}
	return nil, fmt.Errorf("no admin row")
}

func TestLoginUsernameIsCaseSensitiveLikeJava(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("rahasia-panjang"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	auth := &Auth{
		Store:   credentialStore{username: "admin", hash: string(hash)},
		Secret:  []byte(strings.Repeat("s", 32)),
		Limiter: NewRateLimiter(),
		MaxAge:  time.Hour,
	}
	login := func(username string) int {
		body := strings.NewReader(`{"username":"` + username + `","password":"rahasia-panjang"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
		request.RemoteAddr = "203.0.113.5:1234"
		recorder := httptest.NewRecorder()
		auth.Login(recorder, request)
		return recorder.Code
	}
	// findByUsername is exact, so a differently cased or padded username must
	// not authenticate even when the password is right.
	if code := login("Admin"); code != http.StatusUnauthorized {
		t.Errorf("login(Admin) = %d, want 401", code)
	}
	if code := login(" admin"); code != http.StatusUnauthorized {
		t.Errorf("login(\" admin\") = %d, want 401", code)
	}
	if code := login("admin"); code != http.StatusOK {
		t.Errorf("login(admin) = %d, want 200", code)
	}
}

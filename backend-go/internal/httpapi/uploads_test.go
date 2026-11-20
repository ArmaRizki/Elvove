package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/elvove/photography/backend-go/internal/config"
)

// TestUploadsHandlerServesOnlyUploadedFiles pins the rule the handler exists
// for: a browser may read a file the site published under UPLOAD_DIR, and
// nothing else. The original photos live in UPLOAD_DIR/originals and stay
// admin-only, and no request may walk out of UPLOAD_DIR.
//
// This is not a regression test for a past hole: the guard rejects ".." and
// "originals" before anything is opened, and http.Dir would refuse a path that
// leaves its root anyway. It locks the rule so a future rewrite cannot weaken
// it, and it pins one behaviour of serving the checked handle directly: a
// trailing slash on a file answers the file, not a redirect.
//
// The handler is called directly, because the mux cleans the path before it,
// which would hide what the handler itself does with a hostile path.
func TestUploadsHandlerServesOnlyUploadedFiles(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "uploads")
	files := map[string]string{
		"thumbnails/a.webp":                    "thumbnail bytes",
		"previews/a.webp":                      "preview bytes",
		"originals/private.jpg":                "the original, admin only",
		"originals/gallery/token/photo.source": "a client's original",
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	// A file next to UPLOAD_DIR, the neighbour a traversal would reach for.
	if err := os.WriteFile(filepath.Join(parent, "secret.jpg"), []byte("secret"), 0o640); err != nil {
		t.Fatal(err)
	}

	server := &Server{Config: config.Config{UploadDir: root}}
	router := server.uploadsHandler()

	tests := []struct {
		name, target string
		status       int
		body         string
	}{
		{"a published thumbnail is served", "/uploads/thumbnails/a.webp", http.StatusOK, "thumbnail bytes"},
		{"a published preview is served", "/uploads/previews/a.webp", http.StatusOK, "preview bytes"},
		{"the originals folder is not public", "/uploads/originals/private.jpg", http.StatusNotFound, ""},
		{"an encoded path into originals is not public", "/uploads/%6friginals/private.jpg", http.StatusNotFound, ""},
		{"a client's original inside originals is not public", "/uploads/originals/gallery/token/photo.source", http.StatusNotFound, ""},
		{"walking up to a neighbour answers 404", "/uploads/../secret.jpg", http.StatusNotFound, ""},
		{"an encoded walk up answers 404", "/uploads/%2e%2e/secret.jpg", http.StatusNotFound, ""},
		{"a deeply encoded walk up answers 404", "/uploads/thumbnails/%2e%2e/%2e%2e/secret.jpg", http.StatusNotFound, ""},
		{"a missing file answers 404", "/uploads/thumbnails/missing.webp", http.StatusNotFound, ""},
		{"a directory answers 404 rather than a listing", "/uploads/thumbnails/.", http.StatusNotFound, ""},
		{"a clean directory path answers 404 rather than a listing", "/uploads/thumbnails", http.StatusNotFound, ""},
		// http.FileServer redirects this one to "../a.webp"; serving the checked
		// handle answers the photo instead.
		{"a file with a trailing slash is served, not redirected", "/uploads/thumbnails/a.webp/", http.StatusOK, "thumbnail bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			// The mux would clean the path; setting it directly is what a
			// hostile client sends when it bypasses nginx.
			request.URL.Path = test.target
			request.URL.RawPath = ""
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("%s answered %d, want %d", test.target, recorder.Code, test.status)
			}
			if test.body != "" && recorder.Body.String() != test.body {
				t.Fatalf("%s answered %q, want %q", test.target, recorder.Body.String(), test.body)
			}
			if test.body == "" && recorder.Code == http.StatusOK {
				t.Fatalf("%s should not have been served", test.target)
			}
		})
	}
}

// TestUploadsHandlerSendsValidators checks that a cached photo is not sent
// again: the browser keeps a thumbnail for a week, so revalidation has to work.
func TestUploadsHandlerSendsValidators(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "thumbnails"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "thumbnails", "a.webp")
	if err := os.WriteFile(path, []byte("thumbnail bytes"), 0o640); err != nil {
		t.Fatal(err)
	}
	server := &Server{Config: config.Config{UploadDir: root}}
	router := server.uploadsHandler()

	first := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.URL.Path = "/uploads/thumbnails/a.webp"
	router.ServeHTTP(first, request)
	modified := first.Header().Get("Last-Modified")
	if modified == "" {
		t.Fatal("the response has no Last-Modified header")
	}

	second := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.URL.Path = "/uploads/thumbnails/a.webp"
	request.Header.Set("If-Modified-Since", modified)
	router.ServeHTTP(second, request)
	if second.Code != http.StatusNotModified {
		t.Fatalf("a revalidation answered %d, want 304", second.Code)
	}
}

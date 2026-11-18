package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A folder link and a bare id both have to end up as the same folder id, since
// the admin pastes whichever Drive shows them.
func TestFolderIDAcceptsLinksAndIDs(t *testing.T) {
	cases := map[string]string{
		"https://drive.google.com/drive/folders/1a2B3c4D5e6F7g8H9i":       "1a2B3c4D5e6F7g8H9i",
		"https://drive.google.com/drive/u/0/folders/1a2B3c4D5e6F7g8H9i":   "1a2B3c4D5e6F7g8H9i",
		"https://drive.google.com/drive/folders/1a2B3c4D5e6F7g8H9i?hl=id": "1a2B3c4D5e6F7g8H9i",
		" 1a2B3c4D5e6F7g8H9i ":                                "1a2B3c4D5e6F7g8H9i",
		"https://drive.google.com/open?id=1a2B3c4D5e6F7g8H9i": "1a2B3c4D5e6F7g8H9i",
	}
	for input, want := range cases {
		got, err := FolderID(input)
		if err != nil || got != want {
			t.Errorf("FolderID(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"", "   ", "https://example.com/drive", "short"} {
		if _, err := FolderID(input); err == nil {
			t.Errorf("FolderID(%q) should fail", input)
		}
	}
}

// The token is minted once and reused until it nears expiry, so listing a
// hundred files does not mean a hundred token requests.
func TestAccessTokenIsCached(t *testing.T) {
	var tokenCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "token-abc", "expires_in": 3600})
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL

	for i := 0; i < 3; i++ {
		token, err := c.accessToken(context.Background())
		if err != nil || token != "token-abc" {
			t.Fatalf("accessToken = %q, %v", token, err)
		}
	}
	if tokenCalls != 1 {
		t.Errorf("token endpoint called %d times, want 1", tokenCalls)
	}
}

func TestDisabledWithoutCredentials(t *testing.T) {
	if New("", "", "", "", "", "").Enabled() {
		t.Error("a client without credentials must be disabled")
	}
	if !New("id", "secret", "refresh", "", "", "").Enabled() {
		t.Error("a client with all three credentials must be enabled")
	}
}

// A refused refresh token has to surface as an error the handler can report,
// not as an empty file list that looks like an empty folder.
func TestAccessTokenReportsRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant", "error_description": "Token has been expired or revoked."})
	}))
	defer server.Close()
	c := New("id", "secret", "stale", "", "", "")
	c.tokenURL = server.URL

	if _, err := c.accessToken(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("accessToken error = %v, want invalid_grant", err)
	}
}

// EscapeQuery has to survive a folder name with a quote, which would otherwise
// break out of the Drive query string.
func TestEscapeQuery(t *testing.T) {
	if got := escapeQuery(`O'Brien \ photos`); got != `O\'Brien \\ photos` {
		t.Errorf("escapeQuery = %q", got)
	}
}

func TestListImagesFollowsPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		if r.URL.Query().Get("pageToken") == "" {
			json.NewEncoder(w).Encode(map[string]any{
				"nextPageToken": "next",
				"files":         []map[string]any{{"id": "a", "name": "a.jpg", "mimeType": "image/jpeg", "imageMediaMetadata": map[string]any{"width": 6000, "height": 4000}}},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{{"id": "b", "name": "b.jpg", "mimeType": "image/jpeg"}}})
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	files, err := c.ListImages(context.Background(), "folder")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].ID != "a" || files[1].ID != "b" {
		t.Fatalf("files = %+v", files)
	}
	if files[0].Width != 6000 || files[0].Height != 4000 {
		t.Errorf("image metadata lost: %+v", files[0])
	}
}

func TestEnsureFolderCreatesWhenMissing(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Query().Get("q"), "name = 'webp'"):
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{}})
		case r.Method == http.MethodPost:
			created = true
			json.NewEncoder(w).Encode(map[string]any{"id": "new-folder"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	id, err := c.EnsureFolder(context.Background(), "parent", "webp")
	if err != nil || id != "new-folder" || !created {
		t.Fatalf("EnsureFolder = %q, %v (created %v)", id, err, created)
	}
}

func TestEnsureFolderReusesExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{{"id": "existing"}}})
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	id, err := c.EnsureFolder(context.Background(), "parent", "webp")
	if err != nil || id != "existing" {
		t.Fatalf("EnsureFolder = %q, %v", id, err)
	}
}

// A folder the client opens has to be readable by link, and sharing it twice
// must not ask Google for a second permission it would refuse.
func TestShareViewerCreatesOneAnyonePermission(t *testing.T) {
	var creates int
	shared := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
		case r.Method == http.MethodGet:
			permissions := []map[string]any{}
			if shared {
				permissions = append(permissions, map[string]any{"type": "anyone"})
			}
			json.NewEncoder(w).Encode(map[string]any{"permissions": permissions})
		case r.Method == http.MethodPost:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["role"] != "reader" || body["type"] != "anyone" {
				t.Errorf("permission body = %v", body)
			}
			creates++
			shared = true
			json.NewEncoder(w).Encode(map[string]any{"id": "anyoneWithLink"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	for i := 0; i < 2; i++ {
		if err := c.ShareViewer(context.Background(), "folder-1"); err != nil {
			t.Fatalf("ShareViewer = %v", err)
		}
	}
	if creates != 1 {
		t.Errorf("permission created %d times, want 1", creates)
	}
}

func TestAPIFailureIsReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"message":"File not found: abc."}}`)
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	if _, err := c.ListImages(context.Background(), "abc"); err == nil || !strings.Contains(err.Error(), "File not found") {
		t.Fatalf("ListImages error = %v", err)
	}
}

// The client page reads the whole folder tree: a photographer who sorts the
// photos into subfolders must still see them, and the site's own folders (webp,
// foto terpilih, foto edit) must not show up as photos.
func TestListImagesRecursiveWalksSubfoldersAndSkipsOwn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		q := r.URL.Query().Get("q")
		switch {
		case strings.Contains(q, "'root' in parents and trashed = false and mimeType contains 'image/'"):
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{{"id": "top", "name": "top.jpg", "mimeType": "image/jpeg"}}})
		case strings.Contains(q, "'root' in parents and trashed = false and mimeType = 'application/vnd.google-apps.folder'"):
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{
				{"id": "day1", "name": "hari 1"},
				{"id": "webp", "name": "webp"},
			}})
		case strings.Contains(q, "'day1' in parents and trashed = false and mimeType = 'application/vnd.google-apps.folder'"):
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{}})
		case strings.Contains(q, "'day1' in parents"):
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{{"id": "deep", "name": "deep.jpg", "mimeType": "image/jpeg"}}})
		case strings.Contains(q, "'webp' in parents"):
			t.Error("the webp folder must be skipped, not walked")
			json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{}})
		default:
			t.Errorf("unexpected query %s", q)
		}
	}))
	defer server.Close()
	c := New("id", "secret", "refresh", "", "", "")
	c.tokenURL = server.URL + "/token"
	c.apiBase = server.URL

	files, err := c.ListImagesRecursive(context.Background(), "root", "webp", "foto terpilih", "foto edit")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].ID != "top" || files[1].ID != "deep" {
		t.Fatalf("files = %+v", files)
	}
}

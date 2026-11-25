// Package drive is the small part of the Google Drive REST API this site needs:
// list the images in a folder, find or create a subfolder, upload a file, copy
// a file and read one back.
//
// It calls the REST endpoints directly instead of pulling in
// google.golang.org/api, which would add a large dependency tree for five calls.
// Authorization is OAuth on the account that owns the photos: a service account
// has no storage quota of its own, so it can read a shared folder but cannot
// upload into it.
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenURL   = "https://oauth2.googleapis.com/token"
	defaultAPIBase    = "https://www.googleapis.com/drive/v3"
	defaultUploadBase = "https://www.googleapis.com/upload/drive/v3"

	// Scope is the write scope the refresh token has to carry. drive.file would
	// only see files this app created, and the raw folders exist already, so
	// the client page would show nothing.
	Scope = "https://www.googleapis.com/auth/drive"

	folderMime = "application/vnd.google-apps.folder"
)

// Client holds the OAuth credentials and caches the access token they mint. A
// zero-value client (no credentials) is disabled rather than broken, so the
// site keeps working without Drive until the admin sets the credentials.
type Client struct {
	clientID     string
	clientSecret string
	refreshToken string
	http         *http.Client

	// Endpoints are fields so a test can point them at a local server.
	tokenURL   string
	apiBase    string
	uploadBase string

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// New builds the client. Empty endpoints fall back to Google's addresses; they
// are parameters so a test can point the client at a stub.
func New(clientID, clientSecret, refreshToken, tokenURL, apiBase, uploadBase string) *Client {
	pick := func(value, fallback string) string {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return strings.TrimRight(trimmed, "/")
		}
		return fallback
	}
	return &Client{
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		refreshToken: strings.TrimSpace(refreshToken),
		http:         &http.Client{Timeout: 3 * time.Minute},
		tokenURL:     pick(tokenURL, defaultTokenURL),
		apiBase:      pick(apiBase, defaultAPIBase),
		uploadBase:   pick(uploadBase, defaultUploadBase),
	}
}

// Enabled reports whether the three credentials are present.
func (c *Client) Enabled() bool {
	return c != nil && c.clientID != "" && c.clientSecret != "" && c.refreshToken != ""
}

// File is one image in a Drive folder.
type File struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Width    int
	Height   int
}

// FolderID accepts what an admin pastes: a folder link, or the bare id.
func FolderID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("folder id is empty")
	}
	if id := folderInURL(trimmed); id != "" {
		return id, nil
	}
	if bareIDPattern.MatchString(trimmed) {
		return trimmed, nil
	}
	return "", fmt.Errorf("not a Drive folder link or id")
}

var (
	folderPathPattern  = regexp.MustCompile(`/folders/([A-Za-z0-9_-]{10,100})`)
	folderQueryPattern = regexp.MustCompile(`[?&]id=([A-Za-z0-9_-]{10,100})`)
	bareIDPattern      = regexp.MustCompile(`^[A-Za-z0-9_-]{10,100}$`)
)

func folderInURL(value string) string {
	if !strings.Contains(value, "drive.google.com") {
		return ""
	}
	if match := folderPathPattern.FindStringSubmatch(value); match != nil {
		return match[1]
	}
	if match := folderQueryPattern.FindStringSubmatch(value); match != nil {
		return match[1]
	}
	return ""
}

// AccessToken lets the Calendar client use the same connection instead of
// trading a second one with Google: one refresh token, one cached access token.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	return c.accessToken(ctx)
}

// accessToken mints a short-lived token from the refresh token and caches it
// until a minute before it expires.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expiry.Add(-time.Minute)) {
		return c.token, nil
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"refresh_token": {c.refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken      string `json:"access_token"`
		ExpiresIn        int    `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("drive token: %w", err)
	}
	if resp.StatusCode != http.StatusOK || body.AccessToken == "" {
		return "", fmt.Errorf("drive token: %s %s", body.Error, body.ErrorDescription)
	}
	c.token = body.AccessToken
	c.expiry = time.Now().Add(time.Duration(body.ExpiresIn) * time.Second)
	return c.token, nil
}

// do sends one authorized request. A 401 usually means the cached token was
// revoked, so the request is retried once with a fresh one.
func (c *Client) do(ctx context.Context, method, target string, body io.Reader, contentType string, out any) error {
	for attempt := 0; ; attempt++ {
		token, err := c.accessToken(ctx)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, method, target, body)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 && body == nil {
			resp.Body.Close()
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			continue
		}
		if resp.StatusCode >= 300 {
			// The Google message names the actual problem (wrong folder id,
			// permission missing), so it is kept for the log. A 404 is wrapped
			// so callers can tell "gone" from "broken".
			payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
			resp.Body.Close()
			failure := fmt.Errorf("drive %s %s: %s: %s", method, target, resp.Status, strings.TrimSpace(string(payload)))
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("%w: %v", ErrNotFound, failure)
			}
			return failure
		}
		if out == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil
		}
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(out)
	}
}

// ListImages returns the images directly inside a folder, in name order so the
// client page always presents the same sequence.
func (c *Client) ListImages(ctx context.Context, folderID string) ([]File, error) {
	files := make([]File, 0)
	pageToken := ""
	for {
		query := url.Values{
			"q":        {"'" + escapeQuery(folderID) + "' in parents and trashed = false and mimeType contains 'image/'"},
			"fields":   {"nextPageToken,files(id,name,mimeType,imageMediaMetadata(width,height))"},
			"orderBy":  {"name_natural"},
			"pageSize": {"200"},
		}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID                 string `json:"id"`
				Name               string `json:"name"`
				MimeType           string `json:"mimeType"`
				ImageMediaMetadata *struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"imageMediaMetadata"`
			} `json:"files"`
		}
		if err := c.do(ctx, http.MethodGet, c.apiBase+"/files?"+query.Encode(), nil, "", &page); err != nil {
			return nil, err
		}
		for _, f := range page.Files {
			file := File{ID: f.ID, Name: f.Name, MimeType: f.MimeType}
			if f.ImageMediaMetadata != nil {
				file.Width = f.ImageMediaMetadata.Width
				file.Height = f.ImageMediaMetadata.Height
			}
			files = append(files, file)
		}
		if page.NextPageToken == "" {
			return files, nil
		}
		pageToken = page.NextPageToken
	}
}

// ListFolders returns the subfolders of a folder.
func (c *Client) ListFolders(ctx context.Context, folderID string) ([]Info, error) {
	folders := make([]Info, 0)
	pageToken := ""
	for {
		query := url.Values{
			"q":        {"'" + escapeQuery(folderID) + "' in parents and trashed = false and mimeType = '" + folderMime + "'"},
			"fields":   {"nextPageToken,files(id,name)"},
			"orderBy":  {"name_natural"},
			"pageSize": {"200"},
		}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"files"`
		}
		if err := c.do(ctx, http.MethodGet, c.apiBase+"/files?"+query.Encode(), nil, "", &page); err != nil {
			return nil, err
		}
		for _, f := range page.Files {
			folders = append(folders, Info{ID: f.ID, Name: f.Name})
		}
		if page.NextPageToken == "" {
			return folders, nil
		}
		pageToken = page.NextPageToken
	}
}

// ListImagesRecursive lists the images in a folder and in the subfolders below
// it.
//
// The photographer sorts the photos into subfolders, so a flat list would hide
// them. The folders the site keeps for itself (the webp previews, the picked
// photos, the edited photos) are skipped: the client page shows the photos, not
// the site's own copies of them.
func (c *Client) ListImagesRecursive(ctx context.Context, folderID string, skip ...string) ([]File, error) {
	files, err := c.ListImages(ctx, folderID)
	if err != nil {
		return nil, err
	}
	folders, err := c.ListFolders(ctx, folderID)
	if err != nil {
		return nil, err
	}
	for _, folder := range folders {
		if nameIn(folder.Name, skip) {
			continue
		}
		more, err := c.ListImagesRecursive(ctx, folder.ID, skip...)
		if err != nil {
			return nil, err
		}
		files = append(files, more...)
	}
	return files, nil
}

func nameIn(name string, names []string) bool {
	for _, candidate := range names {
		if name == candidate {
			return true
		}
	}
	return false
}

// EnsureFolder returns the id of the named subfolder, creating it when missing.
func (c *Client) EnsureFolder(ctx context.Context, parentID, name string) (string, error) {
	query := url.Values{
		"q":      {"name = '" + escapeQuery(name) + "' and mimeType = '" + folderMime + "' and trashed = false and '" + escapeQuery(parentID) + "' in parents"},
		"fields": {"files(id,name)"},
	}
	var found struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if err := c.do(ctx, http.MethodGet, c.apiBase+"/files?"+query.Encode(), nil, "", &found); err != nil {
		return "", err
	}
	if len(found.Files) > 0 {
		return found.Files[0].ID, nil
	}
	return c.CreateFolder(ctx, parentID, name)
}

func (c *Client) CreateFolder(ctx context.Context, parentID, name string) (string, error) {
	payload := map[string]any{"name": name, "mimeType": folderMime, "parents": []string{parentID}}
	body, _ := json.Marshal(payload)
	var created struct {
		ID string `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, c.apiBase+"/files?fields=id", strings.NewReader(string(body)), "application/json", &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// Copy duplicates one file into a folder, keeping the original bytes — the
// edited photos are made from the full-resolution original, not the preview.
func (c *Client) Copy(ctx context.Context, fileID, parentID, name string) (string, error) {
	payload := map[string]any{"name": name, "parents": []string{parentID}}
	body, _ := json.Marshal(payload)
	var created struct {
		ID string `json:"id"`
	}
	target := c.apiBase + "/files/" + url.PathEscape(fileID) + "/copy?fields=id"
	if err := c.do(ctx, http.MethodPost, target, strings.NewReader(string(body)), "application/json", &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// Download opens the file's bytes. The caller closes the body.
func (c *Client) Download(ctx context.Context, fileID string) (io.ReadCloser, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	target := c.apiBase + "/files/" + url.PathEscape(fileID) + "?alt=media"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		resp.Body.Close()
		return nil, fmt.Errorf("drive download %s: %s: %s", fileID, resp.Status, strings.TrimSpace(string(payload)))
	}
	return resp.Body, nil
}

// Upload stores one file in a folder with a multipart request. The payloads
// here are WebP previews (a few hundred KB), so buffering them is fine.
func (c *Client) Upload(ctx context.Context, parentID, name, mimeType string, data []byte) (string, error) {
	metadata, _ := json.Marshal(map[string]any{"name": name, "parents": []string{parentID}})
	var body strings.Builder
	boundary := "elvove-drive-boundary"
	body.WriteString("--" + boundary + "\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n")
	body.Write(metadata)
	body.WriteString("\r\n--" + boundary + "\r\nContent-Type: " + mimeType + "\r\n\r\n")
	body.Write(data)
	body.WriteString("\r\n--" + boundary + "--\r\n")

	target := c.uploadBase + "/files?uploadType=multipart&fields=id"
	var created struct {
		ID string `json:"id"`
	}
	err := c.do(ctx, http.MethodPost, target, strings.NewReader(body.String()), "multipart/related; boundary="+boundary, &created)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

// escapeQuery quotes a value for a Drive query string.
func escapeQuery(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`)
}

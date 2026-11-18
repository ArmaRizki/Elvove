package drive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// The calls the admin panel needs on top of the file operations: which account
// is connected, and what a folder is.

// About is the Drive account behind the credentials.
type About struct {
	Email string
	Name  string
}

// Account reads the signed-in account, so the panel can show which Google
// account the photos belong to instead of a bare "connected".
func (c *Client) Account(ctx context.Context) (About, error) {
	var body struct {
		User struct {
			EmailAddress string `json:"emailAddress"`
			DisplayName  string `json:"displayName"`
		} `json:"user"`
	}
	target := c.apiBase + "/about?fields=user(emailAddress,displayName)"
	if err := c.do(ctx, http.MethodGet, target, nil, "", &body); err != nil {
		return About{}, err
	}
	return About{Email: body.User.EmailAddress, Name: body.User.DisplayName}, nil
}

// Info is what the panel shows for one folder.
type Info struct {
	ID   string
	Name string
}

// Folder reads a folder's metadata. A folder that is gone answers ErrNotFound,
// which the panel turns into "paste another link".
func (c *Client) Folder(ctx context.Context, folderID string) (Info, error) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	target := c.apiBase + "/files/" + url.PathEscape(folderID) + "?fields=id,name"
	if err := c.do(ctx, http.MethodGet, target, nil, "", &body); err != nil {
		return Info{}, err
	}
	return Info{ID: body.ID, Name: body.Name}, nil
}

// Rename changes the display name of a file or folder.
func (c *Client) Rename(ctx context.Context, fileID, name string) error {
	target := c.apiBase + "/files/" + url.PathEscape(fileID) + "?fields=id"
	payload, _ := json.Marshal(map[string]any{"name": name})
	return c.do(ctx, http.MethodPatch, target, strings.NewReader(string(payload)), "application/json", nil)
}

// ShareViewer makes a file or folder readable by anyone who has its link, so a
// client can open it without asking for access. Sharing a folder covers what is
// inside it. It is safe to call again: an existing "anyone" permission is left
// alone, because Drive refuses a second one.
func (c *Client) ShareViewer(ctx context.Context, fileID string) error {
	base := c.apiBase + "/files/" + url.PathEscape(fileID) + "/permissions"
	var list struct {
		Permissions []struct {
			Type string `json:"type"`
		} `json:"permissions"`
	}
	if err := c.do(ctx, http.MethodGet, base+"?fields=permissions(type)", nil, "", &list); err != nil {
		return err
	}
	for _, p := range list.Permissions {
		if p.Type == "anyone" {
			return nil
		}
	}
	payload, _ := json.Marshal(map[string]any{"role": "reader", "type": "anyone"})
	return c.do(ctx, http.MethodPost, base+"?fields=id", strings.NewReader(string(payload)), "application/json", nil)
}

// Trash moves a file or folder to the trash, contents included. Google keeps it
// for 30 days, so a wrong call is recoverable — unlike Delete.
func (c *Client) Trash(ctx context.Context, fileID string) error {
	target := c.apiBase + "/files/" + url.PathEscape(fileID) + "?fields=id"
	payload, _ := json.Marshal(map[string]any{"trashed": true})
	err := c.do(ctx, http.MethodPatch, target, strings.NewReader(string(payload)), "application/json", nil)
	if errors.Is(err, ErrNotFound) {
		// Already gone: that is the state the caller wanted.
		return nil
	}
	return err
}

// Delete removes a file, or a folder together with its contents. A file that is
// already gone is not an error, so a retry is safe.
func (c *Client) Delete(ctx context.Context, fileID string) error {
	target := c.apiBase + "/files/" + url.PathEscape(fileID)
	err := c.do(ctx, http.MethodDelete, target, nil, "", nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// ErrNotFound is a file or folder that no longer exists. The Drive API answers
// 404 for it, and callers treat that as a normal, recoverable state.
var ErrNotFound = errors.New("drive: not found")

// ExchangeCode turns the authorization code Google sent back into a refresh
// token. It is a method so the token endpoint follows the client's, which is
// what a test needs. The caller stores the token encrypted; it is never logged.
func (c *Client) ExchangeCode(ctx context.Context, redirectURI, code string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
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
		RefreshToken     string `json:"refresh_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("drive code exchange: %w", err)
	}
	if body.RefreshToken == "" {
		return "", fmt.Errorf("drive code exchange: %s %s", body.Error, body.ErrorDescription)
	}
	return body.RefreshToken, nil
}

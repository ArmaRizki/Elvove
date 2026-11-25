// Command driveauth prints a Google OAuth refresh token for the account that
// owns the photo folders.
//
// It is a one-off setup step, run on a machine with a browser:
//
//	GOOGLE_OAUTH_CLIENT_ID=... GOOGLE_OAUTH_CLIENT_SECRET=... go run ./cmd/driveauth
//
// The command prints a consent URL, waits for Google to call back on
// 127.0.0.1:8099, and prints the refresh token. Paste that token into
// GOOGLE_OAUTH_REFRESH_TOKEN on the server; it is never written to the repo.
//
// The consent screen of the Google Cloud project has to be published (even
// unverified): while the app is in "Testing", Google expires refresh tokens
// after seven days, which would break the page every week.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/elvove/photography/backend-go/internal/drive"
	"github.com/elvove/photography/backend-go/internal/gcal"
)

const (
	listenAddr  = "127.0.0.1:8099"
	redirectURI = "http://" + listenAddr + "/callback"
	// Both scopes in one consent: the site writes photos to Drive and bookings
	// to Calendar with the same refresh token, so a token carrying only one of
	// them silently breaks the other feature.
	scope = drive.Scope + " " + gcal.Scope
)

func main() {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		log.Fatal("set GOOGLE_OAUTH_CLIENT_ID and GOOGLE_OAUTH_CLIENT_SECRET first")
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("cannot listen on %s: %v", listenAddr, err)
	}
	defer listener.Close()

	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {scope},
		"access_type":   {"offline"},
		// Without this Google returns no refresh token on a repeat consent.
		"prompt": {"consent"},
	}.Encode()

	fmt.Println("1. Buka tautan ini di browser dan setujui akses:")
	fmt.Println()
	fmt.Println("   " + authURL)
	fmt.Println()
	fmt.Println("2. Setelah disetujui, refresh token akan tercetak di sini.")
	fmt.Printf("\nMenunggu Google memanggil balik di %s ...\n", redirectURI)

	done := make(chan struct{})
	var token string
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Google tidak mengirim kode otorisasi: "+r.URL.Query().Get("error"), http.StatusBadRequest)
			return
		}
		refresh, err := exchange(clientID, clientSecret, code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		token = refresh
		fmt.Fprint(w, "Selesai. Kembali ke terminal untuk melihat token.")
		close(done)
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()

	select {
	case <-done:
	case <-time.After(5 * time.Minute):
		log.Fatal("timeout: tidak ada jawaban dari Google dalam 5 menit")
	}
	_ = server.Close()

	fmt.Println()
	fmt.Println("GOOGLE_OAUTH_REFRESH_TOKEN=" + token)
	fmt.Println()
	fmt.Println("Simpan token ini di .env server, lalu jalankan ulang backend.")
}

func exchange(clientID, clientSecret, code string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
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
		return "", err
	}
	if body.RefreshToken == "" {
		return "", fmt.Errorf("google tidak memberi refresh token: %s %s", body.Error, body.ErrorDescription)
	}
	return body.RefreshToken, nil
}

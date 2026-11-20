package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/elvove/photography/backend-go/internal/drive"
	"github.com/jackc/pgx/v5"
)

// Google Drive, connected from /admin/drive.
//
// The connection lives in drive_settings (one row, like smtp_settings): the
// OAuth client, the refresh token encrypted with SMTP_ENCRYPTION_KEY, and the
// folder every client folder is created in. Because it is stored, the admin
// connects an account in the panel instead of editing .env, and a new booking
// gets its folder without anybody pasting a link.
//
// Without a connection the site keeps working: a booking is saved as before and
// its selection link says the photos are not ready yet.

const googleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"

var (
	errDriveNotConnected = errors.New("google drive belum tersambung")
	errDriveNoRoot       = errors.New("folder induk google drive belum diatur")
)

// driveSettings is one row of drive_settings, with the secrets still encrypted.
type driveSettings struct {
	ClientID        string
	ClientSecretEnc string
	RefreshTokenEnc string
	RootFolderID    string
	RootFolderName  string
	ConnectedEmail  string
	ConnectedAt     string
	// updatedAt stamps the row, so a cached Drive client can tell when the
	// settings were saved again.
	updatedAt string
}

// loadDriveSettings reads the row. A missing row is an empty, unconfigured
// value rather than an error: the panel shows "belum tersambung" instead of
// failing.
func (s *Server) loadDriveSettings(ctx context.Context) (driveSettings, error) {
	var d driveSettings
	err := s.Store.Pool.QueryRow(ctx, `SELECT COALESCE(client_id,''), COALESCE(encrypted_client_secret,''),
			COALESCE(encrypted_refresh_token,''), COALESCE(root_folder_id,''), COALESCE(root_folder_name,''),
			COALESCE(connected_email,''), COALESCE(to_char(connected_at,'YYYY-MM-DD"T"HH24:MI:SS'),''),
			COALESCE(to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS'), '')
		FROM drive_settings WHERE id=1`).
		Scan(&d.ClientID, &d.ClientSecretEnc, &d.RefreshTokenEnc, &d.RootFolderID, &d.RootFolderName, &d.ConnectedEmail, &d.ConnectedAt, &d.updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return driveSettings{}, nil
	}
	return d, err
}

// driveSecrets decrypts the stored client secret. It is separate from the
// refresh token because the secret is needed for the consent step, before a
// token exists.
func (s *Server) driveSecrets(ctx context.Context) (clientID, clientSecret, refreshToken string, err error) {
	settings, err := s.loadDriveSettings(ctx)
	if err != nil {
		return "", "", "", err
	}
	if len(s.Config.SMTPEncryptionKey) < 32 || settings.ClientSecretEnc == "" {
		return settings.ClientID, "", "", nil
	}
	clientSecret, err = decryptSMTPPassword(s.Config.SMTPEncryptionKey, settings.ClientSecretEnc)
	if err != nil {
		return "", "", "", fmt.Errorf("drive client secret: %w", err)
	}
	if settings.RefreshTokenEnc != "" {
		refreshToken, err = decryptSMTPPassword(s.Config.SMTPEncryptionKey, settings.RefreshTokenEnc)
		if err != nil {
			return "", "", "", fmt.Errorf("drive refresh token: %w", err)
		}
	}
	return settings.ClientID, clientSecret, refreshToken, nil
}

// driveSettingsTTL is how long driveClientFor trusts the settings row it last
// read, so a page of thumbnails does not run one query per image.
const driveSettingsTTL = 30 * time.Second

// driveClientFor builds the Drive client for the stored connection, falling back
// to the environment when the database has nothing yet. That keeps a
// deployment configured through GOOGLE_OAUTH_* working until the admin opens
// the panel. The returned client is never nil, so callers only check Enabled().
//
// One client is kept on the Server and rebuilt only when the stored settings
// change: the access token is cached inside the client, so a client per request
// would trade a token with Google on every call.
func (s *Server) driveClientFor(ctx context.Context) *drive.Client {
	s.driveMu.Lock()
	defer s.driveMu.Unlock()
	if time.Since(s.driveCheckedAt) >= driveSettingsTTL {
		settings, err := s.loadDriveSettings(ctx)
		if err != nil {
			log.Printf("drive credentials: %v", err)
		}
		s.cachedDriveSettings = settings
		s.driveCheckedAt = time.Now()
	}
	settings := s.cachedDriveSettings
	if settings.updatedAt == "" || settings.ClientID == "" || settings.ClientSecretEnc == "" || settings.RefreshTokenEnc == "" {
		// Nothing stored (or an unreadable row): the environment decides.
		return s.Drive
	}
	if s.driveFor != settings.updatedAt {
		clientID, secret, refresh, err := s.driveSecrets(ctx)
		if err != nil {
			log.Printf("drive credentials: %v", err)
			return s.Drive
		}
		if clientID != "" && secret != "" && refresh != "" {
			s.drive = drive.New(clientID, secret, refresh, s.Config.GoogleDrive.TokenURL, s.Config.GoogleDrive.APIBase, s.Config.GoogleDrive.UploadBase)
			s.driveFor = settings.updatedAt
		} else {
			return s.Drive
		}
	}
	return s.drive
}

// resetDriveSettingsCache makes the next driveClientFor read the row again, so
// a save or a disconnect shows its effect at once instead of up to 30 s later.
func (s *Server) resetDriveSettingsCache() {
	s.driveMu.Lock()
	s.driveCheckedAt = time.Time{}
	s.driveMu.Unlock()
}

// driveRedirectURI is the address Google calls back on. It has to be typed into
// the Google Cloud console, so the panel shows the exact value.
func (s *Server) driveRedirectURI() string {
	return strings.TrimRight(s.Config.SiteURL, "/") + "/api/admin/drive/callback"
}

func (s *Server) driveAuthURL(clientID string) string {
	query := url.Values{
		"client_id":     {clientID},
		"state":         {s.driveOAuthState()},
		"redirect_uri":  {s.driveRedirectURI()},
		"response_type": {"code"},
		"scope":         {drive.Scope},
		// offline + consent: without both, Google returns no refresh token on a
		// second consent for the same account.
		"access_type": {"offline"},
		"prompt":      {"consent"},
	}
	return googleAuthURL + "?" + query.Encode()
}

// driveOAuthState protects the callback from being driven by somebody else: an
// attacker who sends the admin to the callback with their own code would attach
// their Drive account, and the client photos would land in it. The value is a
// timestamp plus its HMAC, so only this server can produce one.
func (s *Server) driveOAuthState() string {
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	return stamp + "." + hex.EncodeToString(s.driveStateMAC(stamp))
}

func (s *Server) driveStateMAC(stamp string) []byte {
	mac := hmac.New(sha256.New, s.Config.JWTSecret)
	mac.Write([]byte("drive-oauth-state:" + stamp))
	return mac.Sum(nil)
}

// verifyDriveOAuthState accepts a state this server made in the last ten
// minutes: long enough for a consent screen, short enough to be useless later.
func (s *Server) verifyDriveOAuthState(state string) bool {
	stamp, signature, found := strings.Cut(state, ".")
	if !found || stamp == "" || signature == "" {
		return false
	}
	provided, err := hex.DecodeString(signature)
	if err != nil || !hmac.Equal(provided, s.driveStateMAC(stamp)) {
		return false
	}
	issued, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		return false
	}
	return time.Since(time.Unix(issued, 0)) < 10*time.Minute
}

// getDriveSettings answers the panel: whether Drive is connected, which account,
// the root folder and the OAuth client. The client secret and the refresh token
// are never sent back; only whether they are stored.
func (s *Server) getDriveSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.loadDriveSettings(r.Context())
	if err != nil {
		writeError(w, 500, "Pengaturan Google Drive tidak dapat dimuat.")
		return
	}
	writeJSON(w, 200, map[string]any{
		"connected":        settings.ClientID != "" && settings.ClientSecretEnc != "" && settings.RefreshTokenEnc != "",
		"accountEmail":     settings.ConnectedEmail,
		"connectedAt":      nullableString(settings.ConnectedAt),
		"rootFolderId":     settings.RootFolderID,
		"rootFolderName":   settings.RootFolderName,
		"clientId":         settings.ClientID,
		"secretConfigured": settings.ClientSecretEnc != "",
		"tokenConfigured":  settings.RefreshTokenEnc != "",
		"encryptionReady":  len(s.Config.SMTPEncryptionKey) >= 32,
		"redirectUri":      s.driveRedirectURI(),
	})
}

// saveDriveSettings stores the OAuth client. An empty client secret keeps the
// stored one, so the panel can be saved without re-typing secrets.
func (s *Server) saveDriveSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	clientID := strings.TrimSpace(req.ClientID)
	fieldErrors := map[string]string{}
	if clientID == "" {
		fieldErrors["clientId"] = "Client ID wajib diisi."
	} else if len(clientID) > 255 {
		fieldErrors["clientId"] = "Client ID terlalu panjang."
	}
	if len(fieldErrors) > 0 {
		writeValidationError(w, fieldErrors)
		return
	}
	stored, err := s.loadDriveSettings(r.Context())
	if err != nil {
		writeError(w, 500, "Pengaturan Google Drive tidak dapat dimuat.")
		return
	}
	secretEnc := stored.ClientSecretEnc
	if value := strings.TrimSpace(req.ClientSecret); value != "" {
		if len(s.Config.SMTPEncryptionKey) < 32 {
			writeError(w, 409, "SMTP_ENCRYPTION_KEY belum diatur di server, jadi rahasia tidak dapat disimpan.")
			return
		}
		if secretEnc, err = encryptSMTPPassword(s.Config.SMTPEncryptionKey, value); err != nil {
			writeError(w, 500, "Client secret tidak dapat disimpan.")
			return
		}
	}
	if secretEnc == "" {
		writeValidationError(w, map[string]string{"clientSecret": "Client secret wajib diisi."})
		return
	}
	if _, err := s.Store.Exec(r.Context(), `INSERT INTO drive_settings(id,client_id,encrypted_client_secret,created_at,updated_at)
		VALUES(1,$1,$2,NOW(),NOW())
		ON CONFLICT(id) DO UPDATE SET client_id=EXCLUDED.client_id,
			encrypted_client_secret=EXCLUDED.encrypted_client_secret, updated_at=NOW()`, clientID, secretEnc); err != nil {
		writeError(w, 500, "Pengaturan Google Drive tidak dapat disimpan.")
		return
	}
	s.resetDriveSettingsCache()
	s.getDriveSettings(w, r)
}

// startDriveConnect answers the consent address the panel opens in a new tab.
func (s *Server) startDriveConnect(w http.ResponseWriter, r *http.Request) {
	settings, err := s.loadDriveSettings(r.Context())
	if err != nil {
		writeError(w, 500, "Pengaturan Google Drive tidak dapat dimuat.")
		return
	}
	if settings.ClientID == "" || settings.ClientSecretEnc == "" {
		writeError(w, 409, "Isi Client ID dan Client Secret terlebih dahulu.")
		return
	}
	writeJSON(w, 200, map[string]any{"authUrl": s.driveAuthURL(settings.ClientID)})
}

// driveCallback is where Google sends the admin back. It exchanges the code for
// a refresh token, stores it encrypted, reads the account and then returns to
// the panel with the outcome in the query string.
func (s *Server) driveCallback(w http.ResponseWriter, r *http.Request) {
	target := "/admin/drive"
	if !s.verifyDriveOAuthState(r.URL.Query().Get("state")) {
		http.Redirect(w, r, target+"?drive=error&reason=state", http.StatusFound)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if failure := strings.TrimSpace(r.URL.Query().Get("error")); failure != "" || code == "" {
		if failure == "" {
			failure = "no_code"
		}
		http.Redirect(w, r, target+"?drive=error&reason="+url.QueryEscape(failure), http.StatusFound)
		return
	}
	settings, err := s.loadDriveSettings(r.Context())
	if err != nil || settings.ClientID == "" || len(s.Config.SMTPEncryptionKey) < 32 {
		http.Redirect(w, r, target+"?drive=error&reason=not_configured", http.StatusFound)
		return
	}
	secret, err := decryptSMTPPassword(s.Config.SMTPEncryptionKey, settings.ClientSecretEnc)
	if err != nil {
		http.Redirect(w, r, target+"?drive=error&reason=secret", http.StatusFound)
		return
	}
	// The exchange uses the client's own token endpoint, so a test can point it
	// at a stub exactly like the rest of the Drive calls.
	exchange := drive.New(settings.ClientID, secret, "", s.Config.GoogleDrive.TokenURL, s.Config.GoogleDrive.APIBase, s.Config.GoogleDrive.UploadBase)
	refresh, err := exchange.ExchangeCode(r.Context(), s.driveRedirectURI(), code)
	if err != nil {
		log.Printf("drive callback exchange: %v", err)
		http.Redirect(w, r, target+"?drive=error&reason=exchange", http.StatusFound)
		return
	}
	refreshEnc, err := encryptSMTPPassword(s.Config.SMTPEncryptionKey, refresh)
	if err != nil {
		http.Redirect(w, r, target+"?drive=error&reason=encrypt", http.StatusFound)
		return
	}
	// The account name is read now, while the new token is at hand, because it
	// is what tells the admin which Google account is connected.
	probe := drive.New(settings.ClientID, secret, refresh, s.Config.GoogleDrive.TokenURL, s.Config.GoogleDrive.APIBase, s.Config.GoogleDrive.UploadBase)
	about, aboutErr := probe.Account(r.Context())
	if aboutErr != nil {
		log.Printf("drive callback account: %v", aboutErr)
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE drive_settings SET encrypted_refresh_token=$1,
			connected_email=NULLIF($2,''), connected_at=NOW(), updated_at=NOW() WHERE id=1`,
		refreshEnc, about.Email); err != nil {
		log.Printf("drive callback save: %v", err)
		http.Redirect(w, r, target+"?drive=error&reason=save", http.StatusFound)
		return
	}
	s.resetDriveSettingsCache()
	http.Redirect(w, r, target+"?drive=connected", http.StatusFound)
}

// saveDriveRoot stores the parent folder the client folders go in. It reads the
// folder from Drive first, so a typo is caught here.
func (s *Server) saveDriveRoot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RootFolderURL string `json:"rootFolderUrl"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	folderID, err := drive.FolderID(req.RootFolderURL)
	if err != nil {
		writeValidationError(w, map[string]string{"rootFolderUrl": "Tempel link folder Google Drive atau ID foldernya."})
		return
	}
	client := s.driveClientFor(r.Context())
	if !client.Enabled() {
		writeError(w, 409, "Hubungkan akun Google Drive terlebih dahulu.")
		return
	}
	info, err := client.Folder(r.Context(), folderID)
	if err != nil {
		log.Printf("drive root check %s: %v", folderID, err)
		writeError(w, 400, "Folder tidak dapat dibuka. Pastikan link benar dan akun yang tersambung punya akses.")
		return
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE drive_settings SET root_folder_id=$1,
			root_folder_name=NULLIF($2,''), updated_at=NOW() WHERE id=1`, info.ID, info.Name); err != nil {
		writeError(w, 500, "Folder induk tidak dapat disimpan.")
		return
	}
	s.getDriveSettings(w, r)
}

// createDriveRoot makes a folder in My Drive and uses it as the parent, for an
// admin who has no folder yet.
func (s *Server) createDriveRoot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name := sanitizeFolderName(req.Name)
	if name == "" {
		name = "ELVOVE Foto"
	}
	if len(name) > 100 {
		writeValidationError(w, map[string]string{"name": "Nama folder terlalu panjang."})
		return
	}
	client := s.driveClientFor(r.Context())
	if !client.Enabled() {
		writeError(w, 409, "Hubungkan akun Google Drive terlebih dahulu.")
		return
	}
	// "root" is the alias for My Drive in the Drive API.
	folderID, err := client.EnsureFolder(r.Context(), "root", name)
	if err != nil {
		log.Printf("drive root create: %v", err)
		writeError(w, 502, "Folder induk tidak dapat dibuat di Google Drive.")
		return
	}
	if _, err := s.Store.Exec(r.Context(), `UPDATE drive_settings SET root_folder_id=$1,
			root_folder_name=$2, updated_at=NOW() WHERE id=1`, folderID, name); err != nil {
		writeError(w, 500, "Folder induk tidak dapat disimpan.")
		return
	}
	s.getDriveSettings(w, r)
}

// disconnectDrive forgets the token. The client id, the secret and the root
// folder stay, so reconnecting is one consent click instead of re-typing them.
func (s *Server) disconnectDrive(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Store.Exec(r.Context(), `UPDATE drive_settings SET encrypted_refresh_token=NULL,
			connected_email=NULL, connected_at=NULL, updated_at=NOW() WHERE id=1`); err != nil {
		writeError(w, 500, "Google Drive tidak dapat diputuskan.")
		return
	}
	s.resetDriveSettingsCache()
	s.getDriveSettings(w, r)
}

// ---------------------------------------------------------------------------
// Per-booking folders
// ---------------------------------------------------------------------------

// webpFolderName holds the WebP previews the client page serves.
const webpFolderName = "webp"

// pickedFolderName is the subfolder the client's chosen originals are copied
// into. It used to be called "mustedit"; an older folder is renamed the next
// time its booking is touched, so Drive does not end up with two.
const pickedFolderName = "foto terpilih"

// editedFolderName holds the finished edits. It is the folder the invoice links
// to and the only one shared with the client, so the chosen originals stay
// private to the photographer.
const editedFolderName = "foto edit"

// clientFolderName is what a booking's folder is called: the date, then the
// client, so folders sort by session in Drive. Characters Drive rejects are
// replaced, and the booking id keeps two clients with the same name and date
// apart.
func clientFolderName(eventDate, clientName, packageTitle string, bookingID int64) string {
	name := sanitizeFolderName(clientName)
	if name == "" {
		name = "Klien"
	}
	date := strings.TrimSpace(eventDate)
	if date == "" {
		date = "tanpa-tanggal"
	}
	label := date + " - " + name
	if pkg := sanitizeFolderName(packageTitle); pkg != "" {
		label += " (" + pkg + ")"
	}
	// The id is what keeps two clients with a long, similar name apart, so it is
	// added after the cut. Cut on runes: half a character is not a Drive name.
	if runes := []rune(label); len(runes) > 190 {
		label = string(runes[:190])
	}
	return label + fmt.Sprintf(" #%d", bookingID)
}

var folderNoise = regexp.MustCompile(`[^\p{L}\p{N} ._-]+`)

func sanitizeFolderName(value string) string {
	cleaned := folderNoise.ReplaceAllString(strings.TrimSpace(value), " ")
	return strings.Join(strings.Fields(cleaned), " ")
}

// bookingFolders is what a booking's folder tree holds.
type bookingFolders struct {
	FolderID   string
	WebpID     string
	MusteditID string
	EditID     string
}

// ensureBookingFolders creates the booking's folder with its webp, foto terpilih
// and foto edit subfolders and records their ids. It is safe to call twice: a
// folder that is still there is reused, and one somebody deleted from Drive is
// made again.
func (s *Server) ensureBookingFolders(ctx context.Context, bookingID int64) (bookingFolders, error) {
	// The new-booking goroutine and the panel's "Periksa" both run while
	// raw_folder_id is empty, so without this lock each would create its own
	// folder and the row would keep only the last one.
	unlock := s.galleryLock(fmt.Sprintf("folders/%d", bookingID))
	defer unlock()
	client := s.driveClientFor(ctx)
	if !client.Enabled() {
		return bookingFolders{}, errDriveNotConnected
	}
	settings, err := s.loadDriveSettings(ctx)
	if err != nil {
		return bookingFolders{}, err
	}
	if settings.RootFolderID == "" {
		return bookingFolders{}, errDriveNoRoot
	}
	var eventDate, clientName, packageTitle, folderID, webpID, musteditID, editID string
	err = s.Store.Pool.QueryRow(ctx, `SELECT COALESCE(event_date::text,''), COALESCE(name,''), COALESCE(package_title,''),
			COALESCE(raw_folder_id,''),
			COALESCE(folder_webp_id,''), COALESCE(folder_mustedit_id,''), COALESCE(folder_edit_id,'')
		FROM bookings WHERE id=$1`, bookingID).
		Scan(&eventDate, &clientName, &packageTitle, &folderID, &webpID, &musteditID, &editID)
	if err != nil {
		return bookingFolders{}, err
	}
	previousEditID := editID

	// A folder deleted from Drive has to be made again instead of being trusted
	// because its id is still on the row.
	if folderID != "" {
		if _, err := client.Folder(ctx, folderID); err != nil {
			if !errors.Is(err, drive.ErrNotFound) {
				return bookingFolders{}, err
			}
			// Gone from Drive, so the ids under it are stale too.
			folderID, webpID, musteditID, editID = "", "", "", ""
		}
	}
	if folderID == "" {
		folderID, err = client.EnsureFolder(ctx, settings.RootFolderID, clientFolderName(eventDate, clientName, packageTitle, bookingID))
		if err != nil {
			return bookingFolders{}, err
		}
	}
	if err := s.ensureSubfolder(ctx, client, folderID, &webpID, webpFolderName); err != nil {
		return bookingFolders{}, err
	}
	if err := s.ensureSubfolder(ctx, client, folderID, &musteditID, pickedFolderName); err != nil {
		return bookingFolders{}, err
	}
	if err := s.ensureSubfolder(ctx, client, folderID, &editID, editedFolderName); err != nil {
		return bookingFolders{}, err
	}
	// The admin opens these from their own browser, signed in to their own
	// Google account rather than the one the photos belong to, so a private
	// folder answers "you need access". Link access fixes that for the whole
	// tree; the folders only ever appear behind the admin panel and the token
	// page. A failure goes to the log: "Periksa" tries again.
	if err := client.ShareViewer(ctx, folderID); err != nil {
		log.Printf("drive share booking %d folder %s: %v", bookingID, folderID, err)
	}
	// The client opens this folder from the invoice, so it is shared on its own
	// as well: a later move out of the booking folder must not take the access
	// away with it.
	if err := client.ShareViewer(ctx, editID); err != nil {
		log.Printf("drive share booking %d folder %s: %v", bookingID, editID, err)
	}
	folders := bookingFolders{folderID, webpID, musteditID, editID}
	// The invoice link is the edited-photos folder, so it never has to be pasted
	// and the client never sees the private working folders. A link the admin
	// already has is kept, unless the foto edit folder had to be made again and
	// the old link points at a folder that no longer exists.
	// The selection token is minted here too, so a booking taken before links
	// existed gets one as soon as its folder is set up, and the panel can show
	// the link without waiting for the settlement e-mail.
	linkChanged := editID != previousEditID
	if _, err := s.Store.Exec(ctx, `UPDATE bookings SET raw_folder_id=$1::varchar,
			folder_webp_id=NULLIF($2::varchar,''), folder_mustedit_id=NULLIF($3::varchar,''),
			folder_edit_id=NULLIF($4::varchar,''),
			delivery_url=CASE WHEN $5 THEN NULLIF($6::varchar,'')
				ELSE COALESCE(NULLIF(delivery_url,''), NULLIF($6::varchar,'')) END,
			selection_token=COALESCE(NULLIF(selection_token,''), $7::varchar), updated_at=NOW()
		WHERE id=$8`, folderID, webpID, musteditID, editID, linkChanged, driveFolderURL(editID), newSelectionToken(), bookingID); err != nil {
		return folders, err
	}
	return folders, nil
}

// ensureSubfolder keeps a recorded subfolder id honest: one deleted in Drive is
// forgotten so it is made again, and one still carrying its old name is renamed
// instead of being duplicated.
func (s *Server) ensureSubfolder(ctx context.Context, client *drive.Client, parentID string, id *string, name string) error {
	if *id != "" {
		info, err := client.Folder(ctx, *id)
		switch {
		case errors.Is(err, drive.ErrNotFound):
			*id = ""
		case err != nil:
			return err
		case info.Name == name:
			return nil
		default:
			return client.Rename(ctx, *id, name)
		}
	}
	created, err := client.EnsureFolder(ctx, parentID, name)
	if err != nil {
		return err
	}
	*id = created
	return nil
}

// driveFolderURL is the address an admin opens in Drive. An empty id gives an
// empty link, which leaves delivery_url alone.
func driveFolderURL(id string) string {
	if id == "" {
		return ""
	}
	return "https://drive.google.com/drive/folders/" + id
}

// provisionBookingFolders runs the folder creation for a fresh booking in the
// background, so the customer is never kept waiting on Google.
func (s *Server) provisionBookingFolders(bookingID int64) {
	go func() {
		// The goroutine outlives the handler, so a panic here would take the
		// whole process down: the HTTP recover middleware cannot catch it.
		defer func() {
			if v := recover(); v != nil {
				log.Printf("Folder booking #%d panik: %v", bookingID, v)
			}
		}()
		// A booking can be created while Drive is unconfigured; that is not
		// worth an error on every booking, the panel shows it once.
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := s.ensureBookingFolders(ctx, bookingID); err != nil &&
			!errors.Is(err, errDriveNotConnected) && !errors.Is(err, errDriveNoRoot) {
			log.Printf("drive folder booking %d: %v", bookingID, err)
		}
	}()
}

// createBookingDriveFolder is the panel's retry: it creates the folder for one
// booking and answers with the booking, so the panel shows the new folder at
// once.
func (s *Server) createBookingDriveFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := bookingIDFromPath(w, r)
	if !ok {
		return
	}
	if _, err := s.ensureBookingFolders(r.Context(), id); err != nil {
		log.Printf("drive folder booking %d: %v", id, err)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, 404, "Pemesanan tidak ditemukan.")
		case errors.Is(err, errDriveNotConnected):
			writeError(w, 409, "Google Drive belum tersambung. Hubungkan di menu Google Drive.")
		case errors.Is(err, errDriveNoRoot):
			writeError(w, 409, "Folder induk Google Drive belum diatur. Atur di menu Google Drive.")
		default:
			writeError(w, 502, "Folder Google Drive tidak dapat dibuat. Coba lagi.")
		}
		return
	}
	// The folder is checked right before the selection e-mail goes out, so this
	// is the moment to prepare the previews the client is about to open.
	s.warmGalleryPreviews(id)
	booking, err := s.fetchBooking(r, id)
	if err != nil {
		writeError(w, 404, "Pemesanan tidak ditemukan.")
		return
	}
	writeJSON(w, 200, booking)
}

// backfillDriveFolders creates the folder tree for bookings that do not have one
// yet, so connecting Drive also covers bookings taken before it was connected.
// A booking that predates the "foto edit" folder counts as incomplete: running
// this again gives it that folder and the invoice link that goes with it.
func (s *Server) backfillDriveFolders(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id FROM bookings
		WHERE raw_folder_id IS NULL OR raw_folder_id=''
		   OR folder_edit_id IS NULL OR folder_edit_id='' ORDER BY id`)
	if err != nil {
		writeError(w, 500, "Daftar pemesanan tidak dapat dibaca.")
		return
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	created, failed := 0, 0
	// The request context dies when nginx gives up at 60 s, halfway through a
	// long list, so the loop gets its own deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	for _, id := range ids {
		if _, err := s.ensureBookingFolders(ctx, id); err != nil {
			log.Printf("drive backfill booking %d: %v", id, err)
			failed++
			continue
		}
		created++
	}
	writeJSON(w, 200, map[string]any{"created": created, "failed": failed, "total": len(ids)})
}

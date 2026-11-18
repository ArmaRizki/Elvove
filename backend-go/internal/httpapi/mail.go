package httpapi

// SMTP: the saved settings, the dial, the MIME message and the customer mail
// endpoints that send through it.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

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
func (s *Server) smtpDial(r *http.Request, c smtpConfig) (*smtp.Client, net.Conn, error) {
	if c.Authentication && len(s.Config.SMTPEncryptionKey) < 32 {
		return nil, nil, fmt.Errorf("SMTP encryption key is not configured")
	}
	ctx := r.Context()
	address := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	conn, e := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, "tcp", address)
	if e != nil {
		return nil, nil, e
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
			return nil, nil, e
		}
		client, e = smtp.NewClient(tlsConn, c.Host)
	} else {
		client, e = smtp.NewClient(conn, c.Host)
		if e == nil && strings.EqualFold(c.Encryption, "STARTTLS") {
			ok, _ := client.Extension("STARTTLS")
			if !ok {
				client.Close()
				return nil, nil, fmt.Errorf("SMTP server does not support STARTTLS")
			}
			e = client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12})
		}
	}
	if e != nil {
		conn.Close()
		return nil, nil, e
	}
	if c.Authentication {
		password, e := decryptSMTPPassword(s.Config.SMTPEncryptionKey, c.EncryptedPassword)
		if e != nil {
			client.Close()
			return nil, nil, fmt.Errorf("SMTP password could not be decrypted")
		}
		auth := smtp.PlainAuth("", c.Username, password, c.Host)
		if e = client.Auth(auth); e != nil {
			client.Close()
			return nil, nil, e
		}
	}
	return client, conn, nil
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
	client, conn, e := s.smtpDial(req, c)
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
	// The message carries the logo and QRIS images (~180 KB on the wire), which
	// can outlast the 8 s dial budget on a slow uplink.
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	writer, e := client.Data()
	if e != nil {
		return e
	}
	message := buildMIMEMessage(from, to, c.SenderEmail, subject, htmlBody, textBody)
	if _, e = io.WriteString(writer, message); e != nil {
		writer.Close()
		return e
	}
	return writer.Close()
}

// buildMIMEMessage assembles the message sendSMTPMessage hands to the server.
// Date and Message-ID are set explicitly because JavaMail always adds them and
// their absence makes Gmail and spam filters hold the message back.
//
// When there is an HTML body the message is multipart/related: the alternative
// part carries text and HTML, and the images the HTML references by
// "cid:" are attached beside it. Mail clients block remote images by default,
// so the logo and the QR code have to travel with the message.
func buildMIMEMessage(from, to, senderEmail, subject, htmlBody, textBody string) string {
	subjectHeader := mimeWord(subject)
	// Replies go back to the same business inbox the message came from, the
	// same way Java's MimeMessageHelper.setReplyTo does.
	replyTo := ""
	if address := strings.TrimSpace(senderEmail); address != "" {
		replyTo = fmt.Sprintf("Reply-To: %s\r\n", address)
	}
	// WIB, because the JVM this replaced ran with Asia/Jakarta as its default
	// zone and JavaMail stamped Date there. WallClock re-tags as UTC for the
	// database, so the offset is formatted explicitly.
	headers := "Date: " + time.Now().In(jakartaZone).Format(time.RFC1123Z) + "\r\n" +
		"Message-ID: <" + newUploadID() + "@" + messageIDDomain(senderEmail) + ">\r\n"
	if htmlBody == "" {
		return fmt.Sprintf("From: %s\r\nTo: %s\r\n%s%sSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", from, to, replyTo, headers, subjectHeader, base64.StdEncoding.EncodeToString([]byte(textBody)))
	}
	alternative := "elvove-alternative-" + newUploadID()
	related := "elvove-related-" + newUploadID()
	var body strings.Builder
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\n%s%sSubject: %s\r\nMIME-Version: 1.0\r\n", from, to, replyTo, headers, subjectHeader)
	fmt.Fprintf(&body, "Content-Type: multipart/related; boundary=\"%s\"; type=\"multipart/alternative\"\r\n\r\n", related)
	fmt.Fprintf(&body, "--%s\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", related, alternative)
	fmt.Fprintf(&body, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", alternative, base64.StdEncoding.EncodeToString([]byte(textBody)))
	fmt.Fprintf(&body, "\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", alternative, base64.StdEncoding.EncodeToString([]byte(htmlBody)))
	fmt.Fprintf(&body, "\r\n--%s--\r\n", alternative)
	// Only the images this message actually references are attached: an unused
	// part shows up as an attachment in some clients and makes the message
	// heavier for nothing.
	for _, image := range emailInlineImages() {
		if !strings.Contains(htmlBody, emailImageSource(image.ID)) {
			continue
		}
		fmt.Fprintf(&body, "--%s\r\nContent-Type: %s\r\nContent-Transfer-Encoding: base64\r\nContent-ID: <%s>\r\nContent-Disposition: inline; filename=\"%s.png\"\r\n\r\n%s",
			related, image.ContentType, image.ID, image.ID, base64Lines(image.Data))
	}
	fmt.Fprintf(&body, "--%s--\r\n", related)
	return body.String()
}

// messageIDDomain uses the sender's domain, falling back to the site host, so a
// Message-ID is always well formed even when SMTP has no address configured.
func messageIDDomain(senderEmail string) string {
	if _, domain, ok := strings.Cut(strings.TrimSpace(senderEmail), "@"); ok {
		if domain = strings.TrimSpace(domain); domain != "" {
			return domain
		}
	}
	return "elvove.site"
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

// customerMail describes one manually sent customer e-mail: how the panel names
// it, which booking columns record a delivery, which JSON keys report it, and
// when the booking is allowed to receive it. Every column and key here is a
// code constant, never request input.
type customerMail struct {
	Label       string // "konfirmasi", "invoice", "pelunasan"
	NoEmail     string // full sentence when the booking has no client address
	SentColumn  string
	CountColumn string
	SentKey     string
	CountKey    string
	Refuse      func(mailBooking) string // "" when the booking may receive it
	Render      func(mailBooking, string, string, string) (string, string, string)
}

// mailBooking is one booking loaded for a customer e-mail: the template values
// plus the status and payment columns the guards read.
type mailBooking struct {
	bookingEmail
	Status      string
	DownPaidAt  string // YYYY-MM-DD; empty when no payment is recorded
	SettledAt   string // YYYY-MM-DD; empty when the balance is still open
	DeliveryURL string // Google Drive folder with the edited photos
	RawFolder   string // Google Drive folder holding the raw photos
	// The client photo-selection page has its own notice, sent before the
	// settlement one, so the settlement e-mail only asks for money.
	SelectionToken string
	SelectionLimit int
}

// invoiceDetails turns the recorded payment dates, the delivery link and the
// photo-selection link into what the invoice and the settlement notice print.
func (b mailBooking) invoiceDetails(now time.Time) invoiceDetails {
	number := emailInvoiceNumber(b.ID, now.Year())
	return invoiceDetails{
		Number:         number,
		IssuedAt:       now.Format("2006-01-02"),
		DownPaidAt:     b.DownPaidAt,
		SettledAt:      b.SettledAt,
		DeliveryURL:    b.DeliveryURL,
		SelectionLimit: b.SelectionLimit,
	}
}

// ensureSelectionToken gives a booking that predates selection links the token
// its page needs, and returns whatever is stored afterwards. It never replaces a
// token that already exists, so a link already sent keeps working.
func (s *Server) ensureSelectionToken(ctx context.Context, id int64) string {
	if _, err := s.Store.Exec(ctx, `UPDATE bookings SET selection_token=$1
			WHERE id=$2 AND (selection_token IS NULL OR selection_token='')`, newSelectionToken(), id); err != nil {
		log.Printf("selection token booking %d: %v", id, err)
		return ""
	}
	var token string
	if err := s.Store.Pool.QueryRow(ctx, `SELECT COALESCE(selection_token,'') FROM bookings WHERE id=$1`, id).Scan(&token); err != nil {
		log.Printf("selection token booking %d: %v", id, err)
		return ""
	}
	return token
}

// selectionLink is the absolute address of the client page, built from the site
// address the mail is rendered with so the customer opens the same host. The
// token is the only thing that can be missing: the quota is shown on the page
// itself, so a link is never silently dropped from the e-mail.
func (b mailBooking) selectionLink(siteURL string) string {
	if b.SelectionToken == "" {
		return ""
	}
	return strings.TrimRight(emailSiteURL(siteURL), "/") + "/pilih/" + b.SelectionToken
}

// loadMailBooking reads one booking for an e-mail endpoint. A missing row comes
// back as pgx.ErrNoRows so each handler keeps its own not-found wording.
func (s *Server) loadMailBooking(ctx context.Context, id int64) (mailBooking, error) {
	var b mailBooking
	err := s.Store.Pool.QueryRow(ctx, `SELECT id,name,COALESCE(email,''),COALESCE(phone,''),COALESCE(to_char(event_date,'YYYY-MM-DD'),''),COALESCE(to_char(event_time,'HH24:MI'),''),COALESCE(package_title,''),COALESCE(package_price::text,''),COALESCE(event_location,''),status,COALESCE(to_char(down_payment_paid_at,'YYYY-MM-DD'),''),COALESCE(to_char(settlement_paid_at,'YYYY-MM-DD'),''),COALESCE(delivery_url,''),COALESCE(raw_folder_id,''),COALESCE(selection_token,''),`+selectionLimitSQL("bookings")+` FROM bookings WHERE id=$1`, id).
		Scan(&b.ID, &b.Name, &b.Email, &b.Phone, &b.EventDate, &b.EventTime, &b.PackageTitle, &b.PackagePrice, &b.EventLocation, &b.Status, &b.DownPaidAt, &b.SettledAt, &b.DeliveryURL, &b.RawFolder, &b.SelectionToken, &b.SelectionLimit)
	return b, err
}

// sendCustomerMail is the shared body of the manual customer e-mail endpoints:
// guard against a double click, load, refuse when the booking is not eligible,
// render, send, and record the delivery. A failed send leaves the tracking
// columns untouched so the admin can simply retry.
func (s *Server) sendCustomerMail(w http.ResponseWriter, r *http.Request, mail customerMail) {
	id, ok := bookingIDFromPath(w, r)
	if !ok {
		return
	}
	s.mailMu.Lock()
	if s.mailing[id] {
		s.mailMu.Unlock()
		writeError(w, 400, "Email "+mail.Label+" untuk pemesanan ini sedang dikirim. Tunggu hingga proses selesai.")
		return
	}
	s.mailing[id] = true
	s.mailMu.Unlock()
	defer func() {
		s.mailMu.Lock()
		delete(s.mailing, id)
		s.mailMu.Unlock()
	}()
	b, err := s.loadMailBooking(r.Context(), id)
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
	if refusal := mail.Refuse(b); refusal != "" {
		writeError(w, 400, refusal)
		return
	}
	if strings.TrimSpace(b.Email) == "" {
		writeError(w, 400, mail.NoEmail)
		return
	}
	subject, htmlBody, textBody := mail.Render(b, s.mailSenderName(r), s.contactWhatsApp(r), s.Config.SiteURL)
	if err = s.sendSMTPMessage(r, b.Email, subject, htmlBody, textBody); err != nil {
		writeError(w, 400, "Email "+mail.Label+" gagal dikirim. Periksa konfigurasi pada Pengaturan Email, lalu coba lagi.")
		return
	}
	var sentAt time.Time
	var count int
	// The column names are the constants in mail, so this is not string-built SQL
	// from user input.
	record := `UPDATE bookings SET ` + mail.SentColumn + `=NOW(),` + mail.CountColumn +
		`=COALESCE(` + mail.CountColumn + `,0)+1 WHERE id=$1 RETURNING ` + mail.SentColumn + `,` + mail.CountColumn
	if err = s.Store.Pool.QueryRow(r.Context(), record, id).Scan(&sentAt, &count); err != nil {
		writeError(w, 400, "Email "+mail.Label+" terkirim, tetapi status pengiriman gagal disimpan. Silakan periksa data pemesanan.")
		return
	}
	writeJSON(w, 200, map[string]any{
		"message":     "Email " + mail.Label + " berhasil dikirim ke " + b.Email + ".",
		"sentTo":      b.Email,
		mail.SentKey:  wibTimestamp(sentAt),
		mail.CountKey: count,
	})
}

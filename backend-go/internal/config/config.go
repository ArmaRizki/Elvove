package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL       string
	JWTSecret         []byte
	Port              string
	UploadDir         string
	MaxFileSize       int64
	MaxRequest        int64
	Origins           []string
	SMTPEncryptionKey []byte
	// SiteURL is the public site address used in booking e-mails (the header
	// logo and the footer link), mirroring Java's app.mail.site-url.
	SiteURL string
	// MailAdminURL is the base of the admin panel used in the administrator
	// booking notification, mirroring Java's app.mail.admin-url.
	MailAdminURL string
	// AdminUsername, AdminEmail and AdminPassword bootstrap the first admin
	// account when the admins table is empty, like Java's DataInitializer.
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

func Load() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if len([]byte(secret)) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}
	smtpKey := os.Getenv("SMTP_ENCRYPTION_KEY")
	if smtpKey != "" && len([]byte(smtpKey)) < 32 {
		return Config{}, fmt.Errorf("SMTP_ENCRYPTION_KEY must be at least 32 bytes")
	}
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		host := env("DB_HOST", "database")
		port := env("DB_PORT", "5432")
		name := env("DB_NAME", "photographer_portfolio")
		user := env("DB_USERNAME", "postgres")
		pass := os.Getenv("DB_PASSWORD")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", url.QueryEscape(user), url.QueryEscape(pass), host, port, url.PathEscape(name))
	}
	maxFile, err := parseSize(env("UPLOAD_MAX_FILE_SIZE", "50MB"))
	if err != nil {
		return Config{}, fmt.Errorf("UPLOAD_MAX_FILE_SIZE: %w", err)
	}
	maxRequest, err := parseSize(env("UPLOAD_MAX_REQUEST_SIZE", "100MB"))
	if err != nil {
		return Config{}, fmt.Errorf("UPLOAD_MAX_REQUEST_SIZE: %w", err)
	}
	return Config{
		DatabaseURL:       dbURL,
		JWTSecret:         []byte(secret),
		Port:              env("PORT", "8080"),
		UploadDir:         env("UPLOAD_DIR", "/app/uploads"),
		MaxFileSize:       maxFile,
		MaxRequest:        maxRequest,
		Origins:           splitOrigins(env("FRONTEND_URL", "http://localhost:5173")),
		SMTPEncryptionKey: []byte(smtpKey),
		SiteURL:           strings.TrimRight(strings.TrimSpace(env("MAIL_SITE_URL", "https://elvove.site")), "/"),
		MailAdminURL:      strings.TrimSpace(env("MAIL_ADMIN_URL", "https://elvove.site")),
		AdminUsername:     os.Getenv("ADMIN_USERNAME"),
		AdminEmail:        os.Getenv("ADMIN_EMAIL"),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
	}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func parseSize(value string) (int64, error) {
	value = strings.TrimSpace(strings.ToUpper(value))
	multiplier := int64(1)
	switch {
	case strings.HasSuffix(value, "MB"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(value, "MB")
	case strings.HasSuffix(value, "KB"):
		multiplier = 1024
		value = strings.TrimSuffix(value, "KB")
	case strings.HasSuffix(value, "B"):
		value = strings.TrimSuffix(value, "B")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid size %q", value)
	}
	return n * multiplier, nil
}

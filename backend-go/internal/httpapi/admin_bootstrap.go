package httpapi

// The first administrator account, ported from Java's DataInitializer: it is
// created from the environment only while the admins table is empty, so a
// restart never reads, changes or overwrites an existing account.

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/elvove/photography/backend-go/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const minAdminPasswordLength = 12

var adminEmailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Known default / trivially guessable passwords, compared case-insensitively.
var deniedAdminPasswords = map[string]bool{
	"admin123": true, "changeme123!": true, "password": true, "password123": true,
	"admin": true, "12345678": true, "changeme": true,
}

// adminBootstrapError validates the environment values. The messages name the
// variable only; the value is never included.
func adminBootstrapError(username, email, password string) error {
	if username == "" {
		return fmt.Errorf("ADMIN_USERNAME must be set to bootstrap the initial admin account")
	}
	if email == "" {
		return fmt.Errorf("ADMIN_EMAIL must be set to bootstrap the initial admin account")
	}
	if !adminEmailPattern.MatchString(email) {
		return fmt.Errorf("ADMIN_EMAIL is not a valid email address")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("ADMIN_PASSWORD must be set to bootstrap the initial admin account")
	}
	if utf8.RuneCountInString(password) < minAdminPasswordLength {
		return fmt.Errorf("ADMIN_PASSWORD must be at least %d characters long", minAdminPasswordLength)
	}
	if deniedAdminPasswords[strings.ToLower(password)] {
		return fmt.Errorf("ADMIN_PASSWORD is a known default password; choose a strong unique password")
	}
	return nil
}

func BootstrapAdmin(ctx context.Context, db *store.Store, username, email, password string) error {
	var count int64
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM admins`).Scan(&count); err != nil {
		return fmt.Errorf("admin bootstrap: %w", err)
	}
	if count > 0 {
		log.Printf("Admin account already exists; skipping bootstrap")
		return nil
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if err := adminBootstrapError(username, email, password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("admin bootstrap: %w", err)
	}
	now := nowWallClock()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO admins(username,email,password,created_at,updated_at) VALUES($1,$2,$3,$4,$4)`, username, email, string(hash), now); err != nil {
		return fmt.Errorf("admin bootstrap: %w", err)
	}
	log.Printf("Bootstrapped initial admin account %q", username)
	return nil
}

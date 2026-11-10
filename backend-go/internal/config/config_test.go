package config

import (
	"os"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"50MB", 50 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
		{"512KB", 512 * 1024},
		{"2048", 2048},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("parseSize(%q) = %d, %v; want %d", tt.in, got, err, tt.want)
		}
	}
	if _, err := parseSize("0MB"); err == nil {
		t.Fatal("expected zero size to be rejected")
	}
}

func TestSplitOrigins(t *testing.T) {
	got := splitOrigins(" https://elvove.site, http://localhost:5173 ,, ")
	if len(got) != 2 || got[0] != "https://elvove.site" || got[1] != "http://localhost:5173" {
		t.Fatalf("unexpected origins: %#v", got)
	}
}

func TestLoadRequiresStrongSMTPEncryptionKeyWhenProvided(t *testing.T) {
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("SMTP_ENCRYPTION_KEY", "short")
	t.Setenv("DB_URL", "postgres://user:pass@localhost/db?sslmode=disable")
	t.Setenv("UPLOAD_MAX_FILE_SIZE", "50MB")
	t.Setenv("UPLOAD_MAX_REQUEST_SIZE", "100MB")
	if _, err := Load(); err == nil {
		t.Fatal("expected short SMTP key to be rejected")
	}
	t.Setenv("SMTP_ENCRYPTION_KEY", "01234567890123456789012345678901")
	if _, err := Load(); err != nil {
		t.Fatalf("unexpected configuration error: %v", err)
	}
	_ = os.Unsetenv("SMTP_ENCRYPTION_KEY")
}

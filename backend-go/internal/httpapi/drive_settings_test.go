package httpapi

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The booking id at the end is what keeps two clients with a long, similar name
// apart, so it has to survive the cut. The cut itself has to land on a rune
// boundary, or Drive receives a name that is not valid UTF-8.
func TestClientFolderNameKeepsTheIdAfterACut(t *testing.T) {
	name := clientFolderName("2026-11-18", strings.Repeat("é", 200), "Paket Panjang", 12345)
	if !strings.HasSuffix(name, " #12345") {
		t.Errorf("the id was cut off: %q", name)
	}
	if !utf8.ValidString(name) {
		t.Error("the name is not valid UTF-8")
	}
	if runes := []rune(name); len(runes) > 200 {
		t.Errorf("the name is %d runes, want at most 200", len(runes))
	}
}

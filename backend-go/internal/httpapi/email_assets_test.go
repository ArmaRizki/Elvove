package httpapi

import (
	"bytes"
	"encoding/base64"
	"mime"
	"mime/quotedprintable"
	"strings"
	"testing"
)

// The logo and the QRIS code must travel inside the message: every mail client
// blocks remote images until the reader allows them, and a blocked QR code is
// a confirmation the customer cannot pay from.
func TestMessageEmbedsImagesInline(t *testing.T) {
	subject, htmlBody, textBody := renderBookingConfirmation(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")
	message := buildMIMEMessage("Elvove <elvo.vee@gmail.com>", "budi@example.com", "elvo.vee@gmail.com",
		subject, htmlBody, textBody)

	mediaType, params, err := mime.ParseMediaType(headerValue(message, "Content-Type"))
	if err != nil {
		t.Fatalf("Content-Type %q: %v", headerValue(message, "Content-Type"), err)
	}
	if mediaType != "multipart/related" {
		t.Fatalf("Content-Type = %q, want multipart/related", mediaType)
	}

	parts := multipartParts(t, message, params["boundary"])
	// One alternative part plus the two inline images.
	if len(parts) != 3 {
		t.Fatalf("top-level parts = %d, want 3 (alternative + 2 images)", len(parts))
	}

	// The first part is the alternative: text plus HTML.
	alternativeType, alternativeParams, err := mime.ParseMediaType(parts[0].header.Get("Content-Type"))
	if err != nil || alternativeType != "multipart/alternative" {
		t.Fatalf("first part Content-Type = %q, %v", parts[0].header.Get("Content-Type"), err)
	}
	alternatives := multipartParts(t, parts[0].body, alternativeParams["boundary"])
	if len(alternatives) != 2 {
		t.Fatalf("alternative parts = %d, want 2", len(alternatives))
	}
	html := decodedBody(t, alternatives[1])
	for _, cid := range []string{emailLogoCID, emailQRISCID} {
		if !strings.Contains(html, `src="cid:`+cid+`"`) {
			t.Errorf("html does not reference cid:%s", cid)
		}
	}
	if strings.Contains(html, "elvove.site/logo.png") || strings.Contains(html, "elvove.site/qris.png") {
		t.Error("html still points at a remote image URL")
	}

	// The images are attached, decode back to the exact PNG bytes, and carry a
	// Content-ID matching what the HTML references.
	attached := map[string][]byte{}
	for _, part := range parts[1:] {
		contentID := strings.Trim(part.header.Get("Content-ID"), "<>")
		if contentID == "" {
			t.Fatalf("image part without Content-ID")
		}
		if disposition := part.header.Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline") {
			t.Errorf("Content-Disposition = %q, want inline", disposition)
		}
		attached[contentID] = []byte(decodedBody(t, part))
	}
	for _, want := range []struct {
		id   string
		data []byte
	}{
		{emailLogoCID, emailLogoPNG},
		{emailQRISCID, emailQRISPNG},
	} {
		got, ok := attached[want.id]
		if !ok {
			t.Errorf("no attached part for cid:%s", want.id)
			continue
		}
		if !bytes.Equal(got, want.data) {
			t.Errorf("cid:%s payload does not match the embedded PNG (%d vs %d bytes)", want.id, len(got), len(want.data))
		}
		if !bytes.HasPrefix(got, []byte("\x89PNG")) {
			t.Errorf("cid:%s is not a PNG", want.id)
		}
	}
}

// A message that only shows the logo must not drag the QR code along.
func TestOnlyReferencedImagesAreAttached(t *testing.T) {
	subject, htmlBody, textBody := renderBookingReschedule(sampleBooking(), "Elvove", "081234567890", "https://elvove.site")
	message := buildMIMEMessage("Elvove <elvo.vee@gmail.com>", "budi@example.com", "elvo.vee@gmail.com",
		subject, htmlBody, textBody)
	if !strings.Contains(message, "Content-ID: <"+emailLogoCID+">") {
		t.Error("reschedule message is missing the logo")
	}
	if strings.Contains(message, "Content-ID: <"+emailQRISCID+">") {
		t.Error("reschedule message carries the QRIS code it never shows")
	}
}

// A plain-text-only message needs no related part, and must not grow one.
func TestPlainTextMessageHasNoRelatedPart(t *testing.T) {
	message := buildMIMEMessage("a@b.c", "d@e.f", "a@b.c", "Halo", "", "halo")
	if strings.Contains(message, "multipart/related") {
		t.Error("plain-text message should not be multipart/related")
	}
	if !strings.Contains(message, "Content-Type: text/plain; charset=UTF-8") {
		t.Error("plain-text message lost its Content-Type")
	}
}

type mimePart struct {
	header textHeader
	body   string
}

type textHeader map[string]string

func (h textHeader) Get(name string) string { return h[strings.ToLower(name)] }

// multipartParts splits a MIME body on its boundary and decodes the headers of
// each part.
func multipartParts(t *testing.T, message, boundary string) []mimePart {
	t.Helper()
	raw := strings.Split(message, "--"+boundary)
	var parts []mimePart
	for _, chunk := range raw[1:] {
		if strings.HasPrefix(chunk, "--") {
			break
		}
		chunk = strings.TrimPrefix(chunk, "\r\n")
		head, body, ok := strings.Cut(chunk, "\r\n\r\n")
		if !ok {
			t.Fatalf("part has no header/body separator: %q", chunk)
		}
		headers := textHeader{}
		for _, line := range strings.Split(head, "\r\n") {
			name, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			headers[strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
		}
		parts = append(parts, mimePart{header: headers, body: body})
	}
	return parts
}

// decodedBody undoes the base64 (or quoted-printable) transfer encoding of a
// part body.
func decodedBody(t *testing.T, part mimePart) string {
	t.Helper()
	body := strings.TrimSuffix(part.body, "\r\n")
	switch strings.ToLower(part.header.Get("Content-Transfer-Encoding")) {
	case "base64":
		data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
		if err != nil {
			t.Fatalf("base64 decode: %v", err)
		}
		return string(data)
	case "quoted-printable":
		data, err := quotedPrintableDecode(body)
		if err != nil {
			t.Fatalf("quoted-printable decode: %v", err)
		}
		return data
	default:
		return body
	}
}

func quotedPrintableDecode(body string) (string, error) {
	var out bytes.Buffer
	_, err := out.ReadFrom(quotedprintable.NewReader(strings.NewReader(body)))
	return out.String(), err
}

// headerValue returns a top-level header from a raw MIME message.
func headerValue(message, name string) string {
	for _, line := range strings.Split(message, "\r\n") {
		if line == "" {
			return ""
		}
		if got, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(got), name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

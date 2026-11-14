package httpapi

// Images embedded in the booking e-mail. Mail clients block remote images by
// default — Gmail, Outlook and Apple Mail all hide them until the reader allows
// them — so the header logo and the QRIS code are attached to the message and
// referenced by Content-ID instead of by URL. They then render immediately,
// which matters most for the QR code: it is the thing the customer scans.

import (
	_ "embed"
	"encoding/base64"
)

// The files are embedded in the binary so a deployment cannot miss them and
// the image the e-mail carries is the one that shipped with this code.
var (
	//go:embed assets/logo.png
	emailLogoPNG []byte
	//go:embed assets/qris.png
	emailQRISPNG []byte
)

// emailInlineImage is one image attached to the message and referenced from the
// HTML by "cid:<ID>".
type emailInlineImage struct {
	ID          string
	ContentType string
	Data        []byte
}

// The IDs are fixed rather than generated: some clients fail to resolve a
// Content-ID that differs between the HTML and the part, and a constant keeps
// the two in step.
const (
	emailLogoCID = "elvove-logo"
	emailQRISCID = "elvove-qris"
)

// emailInlineImages lists every image the confirmation e-mail references.
func emailInlineImages() []emailInlineImage {
	return []emailInlineImage{
		{ID: emailLogoCID, ContentType: "image/png", Data: emailLogoPNG},
		{ID: emailQRISCID, ContentType: "image/png", Data: emailQRISPNG},
	}
}

// emailImageSource is the src attribute value for an embedded image.
func emailImageSource(id string) string {
	return "cid:" + id
}

// base64Lines encodes data the way MIME wants base64 in a body part: 76
// characters per line, CRLF separated.
func base64Lines(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	const width = 76
	var out []byte
	for len(encoded) > width {
		out = append(out, encoded[:width]...)
		out = append(out, '\r', '\n')
		encoded = encoded[width:]
	}
	out = append(out, encoded...)
	out = append(out, '\r', '\n')
	return string(out)
}

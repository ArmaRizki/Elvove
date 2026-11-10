package main

import (
	"os"

	"github.com/elvove/photography/backend-go/internal/httpapi"
)

func main() {
	subject, html, text := httpapi.PreviewConfirmation()
	_ = os.WriteFile("/out/subject.txt", []byte(subject), 0644)
	_ = os.WriteFile("/out/confirmation.html", []byte(html), 0644)
	_ = os.WriteFile("/out/confirmation.txt", []byte(text), 0644)
}

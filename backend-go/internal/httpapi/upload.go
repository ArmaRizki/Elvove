package httpapi

// Admin image uploads: validate, build the WebP thumbnail and preview
// variants, and clean up when a step fails. The original is not kept: nothing
// could read it, since /uploads/originals/ answers 401.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.Config.MaxRequest)
	if err := r.ParseMultipartForm(s.Config.MaxFileSize); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, 413, uploadTooLargeMessage)
		} else {
			writeError(w, 400, "Upload request is invalid.")
		}
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "Tidak ada berkas foto yang diunggah.")
		return
	}
	defer file.Close()
	path, err := storeUploadedImage(s.Config.UploadDir, s.Config.MaxFileSize, header, file)
	if err != nil {
		status := 400
		message := err.Error()
		if errors.Is(err, errUploadTooLarge) {
			// Spring's multipart parser rejects the part before the controller
			// runs, so its MaxUploadSizeExceededException message is the one
			// Java actually answers with.
			status, message = 413, uploadTooLargeMessage
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, 200, map[string]string{"url": path})
}

const uploadTooLargeMessage = "Ukuran berkas melebihi batas yang diizinkan."

var errUploadTooLarge = errors.New("upload too large")

func storeUploadedImage(root string, max int64, header *multipart.FileHeader, file multipart.File) (string, error) {
	if header.Size < 1 {
		return "", fmt.Errorf("Tidak ada berkas foto yang diunggah.")
	}
	if header.Size > max {
		return "", errUploadTooLarge
	}
	name := filepath.Base(header.Filename)
	dot := strings.LastIndex(name, ".")
	if dot < 0 || dot == len(name)-1 {
		return "", fmt.Errorf("Ekstensi berkas foto tidak valid.")
	}
	ext := strings.ToLower(name[dot:])
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("Format foto harus JPG, PNG, atau WebP.")
	}
	tmp, err := os.CreateTemp("", "elvove-image-")
	if err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	limited := io.LimitReader(file, max+1)
	size, err := io.Copy(tmp, limited)
	if err != nil {
		tmp.Close()
		return "", fmt.Errorf("Upload failed.")
	}
	if size > max {
		tmp.Close()
		return "", errUploadTooLarge
	}
	if err = tmp.Close(); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if !validImageSignature(tmpPath, ext) {
		return "", fmt.Errorf("Berkas bukan gambar JPG, PNG, atau WebP yang valid.")
	}
	if err = os.MkdirAll(filepath.Join(root, "thumbnails"), 0750); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	if err = os.MkdirAll(filepath.Join(root, "previews"), 0750); err != nil {
		return "", fmt.Errorf("Upload failed.")
	}
	id := newUploadID()
	if err = convertWebP(tmpPath, filepath.Join(root, "thumbnails", id+".webp"), 800, 55); err != nil {
		removeUploadVariants(root, "/uploads/"+id+ext)
		return "", fmt.Errorf("Foto gagal diproses. Pastikan berkas adalah gambar yang valid.")
	}
	if err = convertWebP(tmpPath, filepath.Join(root, "previews", id+".webp"), 1200, 65); err != nil {
		removeUploadVariants(root, "/uploads/"+id+ext)
		return "", fmt.Errorf("Foto gagal diproses. Pastikan berkas adalah gambar yang valid.")
	}
	return "/uploads/thumbnails/" + id + ".webp", nil
}
func validImageSignature(path, ext string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var header [12]byte
	n, _ := file.Read(header[:])
	jpeg := n >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff
	png := n >= 8 && header[0] == 0x89 && string(header[1:4]) == "PNG" && header[4] == 0x0d && header[5] == 0x0a && header[6] == 0x1a && header[7] == 0x0a
	webp := n >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP"
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg
	case ".png":
		return png
	case ".webp":
		return webp
	}
	return false
}
func newUploadID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
func convertWebP(source, destination string, width, quality int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cwebp", "-quiet", "-resize", strconv.Itoa(width), "0", "-q", strconv.Itoa(quality), source, "-o", destination)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("webp conversion failed: %s", strings.TrimSpace(string(out)))
	}
	info, err := os.Stat(destination)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("webp output missing")
	}
	return nil
}

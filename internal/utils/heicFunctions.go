package utils

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrium/goheif"
)

// Checks if the MIME type indicates a HEIC or HEIF image format.
func IsHeifLike(mimeType string) bool {
	t := strings.ToLower(mimeType)
	return strings.Contains(t, "heic") || strings.Contains(t, "heif")
}

// Converts HEIC/HEIF image data to JPEG format.
// Returns the JPEG-encoded data or an error if conversion fails.
func ConvertHeicToJpeg(input []byte) ([]byte, error) {
	img, err := goheif.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

func ConvertIfHeic(name, mime string, data []byte) (string, string, []byte) {
	if !IsHeifLike(mime) {
		return name, mime, data
	}

	log.Printf("Converting HEIC to JPEG: %s", name)
	jpeg, err := ConvertHeicToJpeg(data)
	if err != nil {
		log.Printf("HEIC conversion failed for %s: %v", name, err)
		return name, mime, data
	}

	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext) + ".jpg"
	}

	return name, "image/jpeg", jpeg
}

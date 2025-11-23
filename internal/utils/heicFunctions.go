package utils

import (
	"bytes"
	"fmt"
	"image/jpeg"
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

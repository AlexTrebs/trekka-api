package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrium/goheif"
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

// Checks if the MIME type indicates a HEIC or HEIF image format.
func IsHeifLike(mimeType string) bool {
	t := strings.ToLower(mimeType)
	return strings.Contains(t, "heic") || strings.Contains(t, "heif")
}

// Converts HEIC/HEIF image data to JPEG format with proper orientation handling.
// Returns the JPEG-encoded data or an error if conversion fails.
func ConvertHeicToJpeg(input []byte) ([]byte, error) {
	img, err := goheif.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC: %w", err)
	}

	// Apply EXIF orientation if present
	oriented := applyOrientation(img, input)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, oriented, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// Reads EXIF orientation and applies correct transformations to the image
func applyOrientation(img image.Image, input []byte) image.Image {
	// Try to extract EXIF data
	x, err := exif.Decode(bytes.NewReader(input))
	if err != nil {
		log.Printf("[HEIC] No EXIF data found or failed to parse: %v", err)
		return img
	}

	// Get orientation tag
	orientTag, err := x.Get(exif.Orientation)
	if err != nil {
		return img
	}

	orient, err := orientTag.Int(0)
	if err != nil {
		log.Printf("[HEIC] Failed to read orientation value: %v", err)
		return img
	}

	// Apply orientation transformations
	// EXIF orientation values: 1=normal, 2=flip-h, 3=180, 4=flip-v, 5=transpose, 6=270, 7=transverse, 8=90
	switch orient {
	case 1:
		return img
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	default:
		log.Printf("[HEIC] Unknown orientation value: %d", orient)
		return img
	}
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

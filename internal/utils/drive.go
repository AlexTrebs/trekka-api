package utils

import (
	"strings"

	"google.golang.org/api/drive/v3"
)

func IsImage(file *drive.File) bool {
	if file == nil {
		return false
	}
	return strings.HasPrefix(file.MimeType, "image/")
}

package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"trekka-api/internal/models"
)

// Extracts GPS coordinates and metadata from MP4 video data using exiftool
func ExtractMP4Data(videoData []byte) (models.Coordinates, string, []float64, error) {
	// Use exiftool to extract metadata from MP4
	cmd := exec.Command("exiftool", "-n", "-GPSLatitude", "-GPSLongitude", "-CreateDate", "-ImageWidth", "-ImageHeight", "-")
	cmd.Stdin = bytes.NewReader(videoData)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return models.Coordinates{}, "", nil, fmt.Errorf("exiftool failed: %w (output: %s)", err, string(output))
	}

	var coords models.Coordinates
	var timestamp string
	var resolution []float64
	foundGPS := false

	// Parse exiftool output
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if strings.Contains(line, "GPS Latitude") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				latStr := strings.TrimSpace(parts[1])
				lat, err := strconv.ParseFloat(latStr, 64)
				if err == nil {
					coords.Lat = fmt.Sprintf("%.6f", lat)
					foundGPS = true
				}
			}
		} else if strings.Contains(line, "GPS Longitude") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				lngStr := strings.TrimSpace(parts[1])
				lng, err := strconv.ParseFloat(lngStr, 64)
				if err == nil {
					coords.Lng = fmt.Sprintf("%.6f", lng)
				}
			}
		} else if strings.Contains(line, "Create Date") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				timestamp = strings.TrimSpace(parts[1])
				// Convert from "YYYY:MM:DD HH:MM:SS" to ISO 8601 format
				timestamp = regexp.MustCompile(`^(\d{4}):(\d{2}):(\d{2})`).ReplaceAllString(timestamp, "$1-$2-$3")
			}
		} else if strings.Contains(line, "Image Width") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				widthStr := strings.TrimSpace(parts[1])
				width, err := strconv.ParseFloat(widthStr, 64)
				if err == nil && len(resolution) == 0 {
					resolution = []float64{width, 0}
				}
			}
		} else if strings.Contains(line, "Image Height") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				heightStr := strings.TrimSpace(parts[1])
				height, err := strconv.ParseFloat(heightStr, 64)
				if err == nil {
					if len(resolution) == 0 {
						resolution = []float64{0, height}
					} else {
						resolution[1] = height
					}
				}
			}
		}
	}

	if !foundGPS {
		return models.Coordinates{}, timestamp, resolution, fmt.Errorf("no GPS data found in MP4")
	}

	return coords, timestamp, resolution, nil
}

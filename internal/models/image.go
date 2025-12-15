package models

import "time"

type Coordinates struct {
	Lng string `firestore:"lng,omitempty" json:"lng,omitempty"`
	Lat string `firestore:"lat,omitempty" json:"lat,omitempty"`
}

type CacheEntry struct {
	SignedURL   string
	ContentType string
	GeoLocation string
	FileName    string
	Expires     time.Time
}

type ImageRequest struct {
	Id       string
	FileName string
}

type ImageMetadata struct {
	Id            string      `firestore:"id,omitempty"`
	FileName      string      `firestore:"fileName"`
	ContentType   string      `firestore:"contentType"`
	Coordinates   Coordinates `firestore:"coordinates,omitempty"`
	StoragePath   string      `firestore:"storagePath"`
	GeoLocation   string      `firestore:"geoLocation,omitempty"`   // Format: "City, Country"
	FormattedDate string      `firestore:"formattedDate,omitempty"` // Format: "Wednesday, 15 January 2025, 14:30"
	Resolution    []float64   `firestore:"resolution,omitempty"`    // Format: [width, height]
	TakenAt       time.Time   `firestore:"takenAt,omitempty"`       // Actual photo capture time from EXIF
	CreatedAt     time.Time   `firestore:"createdAt,omitempty"`     // When record was created
	UpdatedAt     time.Time   `firestore:"updatedAt,omitempty"`     // When record was updated
}

type ImageResponse struct {
	FileName    string      `json:"fileName"`
	ContentType string      `json:"contentType"`
	GeoLocation string      `json:"geoLocation,omitempty"`
	Coordinates Coordinates `json:"coordinates,omitzero"`
	Size        int         `json:"size"`
}

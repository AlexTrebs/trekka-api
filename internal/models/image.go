package models

import "time"

type Coordinates struct {
	Lng string `firestore:"lng,omitempty" json:"lng,omitempty"`
	Lat string `firestore:"lat,omitempty" json:"lat,omitempty"`
}

type CacheEntry struct {
	Data        []byte
	ContentType string
	GeoLocation string
	FileName    string
	Expires     time.Time
}

type ImageRequest struct {
	FileName string
}

type ImageMetadata struct {
	ID          string      `firestore:"id,omitempty"`
	FileName    string      `firestore:"fileName"`
	ContentType string      `firestore:"contentType"`
	Coordinates Coordinates `firestore:"coordinates,omitempty"`
	StoragePath string      `firestore:"storagePath"`
	GeoLocation string      `firestore:"geoLocation,omitempty"`
	CreatedAt   time.Time   `firestore:"createdAt,omitempty"`
	UpdatedAt   time.Time   `firestore:"updatedAt,omitempty"`
}

type ImageResponse struct {
	FileName    string      `json:"fileName"`
	ContentType string      `json:"contentType"`
	GeoLocation string      `json:"geoLocation,omitempty"`
	Coordinates Coordinates `json:"coordinates,omitzero"`
	Size        int         `json:"size"`
}

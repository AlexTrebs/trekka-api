package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// Handles Drive-related metadata extraction and downloading.
type DriveClient struct {
	client       *drive.Service
	rateLimitMu  sync.Mutex
	lastCallTime time.Time
}

// Creates a DriveFileService with 3-second rate limiting.
func NewDriveClient(client *drive.Service) *DriveClient {
	return &DriveClient{
		client:       client,
		lastCallTime: time.Now().Add(-3 * time.Second), // Allow first call immediately
	}
}

// Ensures at least 3 seconds between Drive API calls to avoid rate limiting.
func (d *DriveClient) waitForRateLimit() {
	d.rateLimitMu.Lock()
	defer d.rateLimitMu.Unlock()

	const minDelay = 5 * time.Second
	elapsed := time.Since(d.lastCallTime)
	if elapsed < minDelay {
		time.Sleep(minDelay - elapsed)
	}
	d.lastCallTime = time.Now()
}

// Find looks up a Drive file by exact name inside a folder with retry logic.
func (d *DriveClient) Find(ctx context.Context, folderID, name string) (*drive.File, error) {
	if d.client == nil {
		return nil, fmt.Errorf("drive client is nil")
	}

	const maxRetries = 3
	backoff := 5 * time.Second

	// Escape single quotes in folder ID and name to prevent query injection
	// Per Drive API docs, single quotes should be escaped by doubling them
	escapedFolderID := strings.ReplaceAll(folderID, "'", "\\'")
	escapedName := strings.ReplaceAll(name, "'", "\\'")
	q := fmt.Sprintf("'%s' in parents and name='%s' and trashed=false", escapedFolderID, escapedName)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		d.waitForRateLimit()

		list, err := d.client.Files.List().Context(ctx).
			Q(q).
			Fields("files(id, name, mimeType, size, createdTime, modifiedTime, imageMediaMetadata, videoMediaMetadata)").
			Do()
		if err != nil {
			// Check for rate limit errors using proper type assertion
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) && (apiErr.Code == 403 || apiErr.Code == 429) {
				if attempt < maxRetries {
					sleepDuration := backoff * time.Duration(1<<uint(attempt))
					time.Sleep(sleepDuration)
					continue
				}
			}
			return nil, err
		}

		if len(list.Files) == 0 {
			return nil, fmt.Errorf("file not found in drive: %s", name)
		}

		return list.Files[0], nil
	}

	return nil, fmt.Errorf("failed to find file after %d retries", maxRetries)
}

// Downloads the file content from Google Drive with exponential backoff retry.
func (d *DriveClient) DownloadBytes(ctx context.Context, id string) ([]byte, error) {
	const maxRetries = 5
	backoff := 5 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("[DriveClient] Retry attempt %d/%d for file %s\n", attempt, maxRetries, id)
		}

		d.waitForRateLimit()

		fmt.Printf("[DriveClient] Making download request for file %s\n", id)
		resp, err := d.client.Files.Get(id).Context(ctx).Download()
		if err != nil {
			fmt.Printf("[DriveClient] Download request failed: %v\n", err)
			// Check for rate limit errors using proper type assertion
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) && (apiErr.Code == 403 || apiErr.Code == 429) {
				if attempt < maxRetries {
					// Exponential backoff: 5s, 10s, 20s, 40s, 80s
					sleepDuration := backoff * time.Duration(1<<uint(attempt))
					fmt.Printf("[DriveClient] Rate limited (HTTP %d), sleeping for %v\n", apiErr.Code, sleepDuration)
					time.Sleep(sleepDuration)
					continue // Retry
				}
			}
			return nil, err
		}
		defer resp.Body.Close()

		fmt.Printf("[DriveClient] Reading response body for file %s\n", id)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[DriveClient] Failed to read response body: %v\n", err)
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		fmt.Printf("[DriveClient] Successfully downloaded %d bytes for file %s\n", len(data), id)
		return data, nil
	}

	return nil, fmt.Errorf("failed to download file after %d retries", maxRetries)
}

// Lists all files in the specified Drive folder (paginated) with retry logic.
func (d *DriveClient) ListFilesInFolder(ctx context.Context, folderID string) ([]*drive.File, error) {
	if d.client == nil {
		return nil, fmt.Errorf("drive client is nil")
	}

	var allFiles []*drive.File
	pageToken := ""

	// Escape single quotes in folder ID to prevent query injection
	escapedFolderID := strings.ReplaceAll(folderID, "'", "\\'")
	query := fmt.Sprintf("'%s' in parents and trashed=false", escapedFolderID)

	for {
		const maxRetries = 3
		backoff := 5 * time.Second
		var fileList *drive.FileList
		var err error

		// Retry logic for each page
		for attempt := 0; attempt <= maxRetries; attempt++ {
			d.waitForRateLimit()

			call := d.client.Files.List().
				Context(ctx).
				Q(query).
				Fields("nextPageToken, files(id, name, mimeType, size, createdTime, modifiedTime, imageMediaMetadata, videoMediaMetadata)").
				PageSize(1000)

			if pageToken != "" {
				call = call.PageToken(pageToken)
			}

			fileList, err = call.Do()
			if err != nil {
				// Check for rate limit errors using proper type assertion
				var apiErr *googleapi.Error
				if errors.As(err, &apiErr) && (apiErr.Code == 403 || apiErr.Code == 429) {
					if attempt < maxRetries {
						sleepDuration := backoff * time.Duration(1<<uint(attempt))
						time.Sleep(sleepDuration)
						continue
					}
				}
				return nil, fmt.Errorf("list files failed: %w", err)
			}
			break // Success, exit retry loop
		}

		if err != nil {
			return nil, fmt.Errorf("list files failed after retries: %w", err)
		}

		allFiles = append(allFiles, fileList.Files...)

		if fileList.NextPageToken == "" {
			break
		}
		pageToken = fileList.NextPageToken
	}

	return allFiles, nil
}

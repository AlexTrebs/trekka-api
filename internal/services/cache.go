package services

import (
	"sync"
	"time"

	"trekka-api/internal/models"
)

type CacheService struct {
	cache           map[string]*models.CacheEntry
	mu              sync.RWMutex
	ttl             time.Duration
	cleanupInterval time.Duration
}

func NewCacheService(ttl, cleanupInterval time.Duration) *CacheService {
	cs := &CacheService{
		cache:           make(map[string]*models.CacheEntry),
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
	}

	// Start cleanup goroutine
	go cs.cleanupExpired()

	return cs
}

// Retrieves a cache entry by key, returning nil if not found or expired.
func (cs *CacheService) Get(key string) (*models.CacheEntry, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	entry, ok := cs.cache[key]
	if !ok {
		return nil, false
	}

	if entry.Expires.Before(time.Now()) {
		return nil, false
	}

	return entry, true
}

// Stores data in the cache with the specified key and metadata.
// The entry will expire after the configured TTL.
func (cs *CacheService) Set(key string, data []byte, contentType, geoLocation, fileName string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cache[key] = &models.CacheEntry{
		Data:        data,
		ContentType: contentType,
		GeoLocation: geoLocation,
		FileName:    fileName,
		Expires:     time.Now().Add(cs.ttl),
	}
}

// Periodically removes expired entries from the cache.
// This runs in a background goroutine started by NewCacheService.
func (cs *CacheService) cleanupExpired() {
	ticker := time.NewTicker(cs.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		cs.mu.Lock()
		for k, v := range cs.cache {
			if v.Expires.Before(now) {
				delete(cs.cache, k)
			}
		}
		cs.mu.Unlock()
	}
}

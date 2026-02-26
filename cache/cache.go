// Package cache provides a thread-safe in-memory cache that is persisted to a
// JSON file (rooam_cache.json) after every refresh cycle.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheFile is the name of the on-disk cache file written to the install directory.
const CacheFile = "rooam_cache.json"

// Data holds all cached POSitouch data.
type Data struct {
	LastUpdated time.Time                `json:"last_updated"`
	CostCenters []map[string]interface{} `json:"cost_centers"`
	Tenders     []map[string]interface{} `json:"tenders"`
	Employees   []map[string]interface{} `json:"employees"`
	Tables      []map[string]interface{} `json:"tables"`
	OrderTypes  []map[string]interface{} `json:"order_types"`
}

// Cache is a thread-safe in-memory store backed by a JSON file.
type Cache struct {
	mu         sync.RWMutex
	data       Data
	installDir string
}

// New creates a new Cache that will persist its data to installDir/rooam_cache.json.
func New(installDir string) *Cache {
	return &Cache{installDir: installDir}
}

// Update atomically replaces all cached data and flushes to disk.
func (c *Cache) Update(d Data) error {
	d.LastUpdated = time.Now().UTC()

	c.mu.Lock()
	c.data = d
	c.mu.Unlock()

	return c.save(d)
}

// Get returns a copy of the current cached data.
func (c *Cache) Get() Data {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// save writes the cache data to rooam_cache.json inside the install directory.
func (c *Cache) save(d Data) error {
	path := filepath.Join(c.installDir, CacheFile)
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshalling: %w", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("cache: writing %s: %w", path, err)
	}
	return nil
}

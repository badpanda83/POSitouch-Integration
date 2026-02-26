// Package cache provides a thread-safe in-memory cache with JSON file persistence.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// CachedData holds all POSitouch data retrieved during a pull cycle.
type CachedData struct {
	LastUpdated time.Time              `json:"last_updated"`
	CostCenters []positouch.CostCenter `json:"cost_centers"`
	Tenders     []positouch.Tender     `json:"tenders"`
	Employees   []positouch.Employee   `json:"employees"`
	Tables      []positouch.Table      `json:"tables"`
	OrderTypes  []positouch.OrderType  `json:"order_types"`
}

// Cache is a thread-safe in-memory store backed by a JSON file.
type Cache struct {
	mu       sync.RWMutex
	data     *CachedData
	filePath string
}

// NewCache creates a new Cache that persists to filePath.
func NewCache(filePath string) *Cache {
	return &Cache{filePath: filePath}
}

// Update replaces the in-memory data and writes it to the JSON file atomically.
func (c *Cache) Update(data *CachedData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}

	// Write to a temp file then rename for atomicity
	tmp := c.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return fmt.Errorf("cache: write temp file: %w", err)
	}
	if err := os.Rename(tmp, c.filePath); err != nil {
		return fmt.Errorf("cache: rename cache file: %w", err)
	}

	c.mu.Lock()
	c.data = data
	c.mu.Unlock()
	return nil
}

// Get returns the current cached data (may be nil if never populated).
func (c *Cache) Get() *CachedData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// Load reads the JSON cache file from disk and populates the in-memory store.
// Returns nil if the file does not exist (not an error on first run).
func (c *Cache) Load() error {
	raw, err := os.ReadFile(c.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cache: read file: %w", err)
	}
	var data CachedData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("cache: parse file: %w", err)
	}
	c.mu.Lock()
	c.data = &data
	c.mu.Unlock()
	return nil
}

// Package cache provides a thread-safe in-memory store for POSitouch data
// that is also persisted to a JSON file on disk.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const cacheFileName = "rooam_cache.json"

// CostCenter represents a POSitouch cost center.
type CostCenter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Tender represents a POSitouch payment type / tender.
type Tender struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Employee represents a POSitouch employee.
type Employee struct {
	ID        int    `json:"id"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Type      int    `json:"type"`
	MagCardID int    `json:"mag_card_id"`
}

// Table represents a unique table entry from the check header file.
type Table struct {
	TableNumber int `json:"table_number"`
	CostCenter  int `json:"cost_center"`
}

// OrderType represents a POSitouch menu / order type.
type OrderType struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	FFOrd int    `json:"ff_ord_t"`
}

// Data holds all cached POSitouch data.
type Data struct {
	CostCenters []CostCenter `json:"cost_centers"`
	Tenders     []Tender     `json:"tenders"`
	Employees   []Employee   `json:"employees"`
	Tables      []Table      `json:"tables"`
	OrderTypes  []OrderType  `json:"order_types"`
	LastUpdated time.Time    `json:"last_updated"`
}

// Cache is a thread-safe in-memory store that also persists to disk.
type Cache struct {
	mu       sync.RWMutex
	data     Data
	filePath string
}

// New creates a new Cache that persists to cacheDir/rooam_cache.json.
func New(cacheDir string) *Cache {
	return &Cache{
		filePath: filepath.Join(cacheDir, cacheFileName),
	}
}

// LoadFromDisk reads the JSON cache file into memory.
// If the file doesn't exist, no error is returned — the cache starts empty.
func (c *Cache) LoadFromDisk() error {
	data, err := os.ReadFile(c.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cache: reading %s: %w", c.filePath, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := json.Unmarshal(data, &c.data); err != nil {
		return fmt.Errorf("cache: parsing %s: %w", c.filePath, err)
	}
	return nil
}

// Update replaces the in-memory data and persists it to disk atomically.
func (c *Cache) Update(d Data) error {
	d.LastUpdated = time.Now()
	c.mu.Lock()
	c.data = d
	c.mu.Unlock()
	return c.saveToDisk()
}

// Get returns a snapshot of the current cached data.
func (c *Cache) Get() Data {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// saveToDisk writes the current data to the JSON file.
func (c *Cache) saveToDisk() error {
	c.mu.RLock()
	data, err := json.MarshalIndent(c.data, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("cache: marshalling: %w", err)
	}
	// Write to a temporary file then rename for atomicity.
	tmp := c.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("cache: writing temp file: %w", err)
	}
	if err := os.Rename(tmp, c.filePath); err != nil {
		return fmt.Errorf("cache: renaming cache file: %w", err)
	}
	return nil
}

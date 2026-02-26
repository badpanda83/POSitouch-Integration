// Package cache provides an in-memory cache for POSitouch data that is
// periodically persisted to a JSON file on disk.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// CacheFileName is the name of the JSON cache file written to the install dir.
const CacheFileName = "rooam_cache.json"

// Data is the top-level structure written to rooam_cache.json.
type Data struct {
	LastUpdated time.Time               `json:"last_updated"`
	Status      string                  `json:"status"`
	CostCenters []positouch.CostCenter  `json:"cost_centers"`
	Tenders     []positouch.Tender      `json:"tenders"`
	Employees   []positouch.Employee    `json:"employees"`
	Tables      []positouch.Table       `json:"tables"`
	OrderTypes  []positouch.OrderType   `json:"order_types"`
	Errors      []string                `json:"errors"`
}

// Cache is a thread-safe in-memory store backed by a JSON file.
type Cache struct {
	mu         sync.RWMutex
	data       Data
	installDir string
}

// New creates an empty Cache that will persist to installDir/rooam_cache.json.
func New(installDir string) *Cache {
	return &Cache{
		installDir: installDir,
		data: Data{
			CostCenters: []positouch.CostCenter{},
			Tenders:     []positouch.Tender{},
			Employees:   []positouch.Employee{},
			Tables:      []positouch.Table{},
			OrderTypes:  []positouch.OrderType{},
			Errors:      []string{},
		},
	}
}

// Update replaces the cached data and persists it to disk.
func (c *Cache) Update(d Data) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = d
	return c.persist()
}

// Get returns a snapshot of the current cached data.
func (c *Cache) Get() Data {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// persist writes the current in-memory data to the JSON cache file.
// Caller must hold c.mu (write lock).
func (c *Cache) persist() error {
	path := filepath.Join(c.installDir, CacheFileName)
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("cache: write %s: %w", path, err)
	}
	return nil
}

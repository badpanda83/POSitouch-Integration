// Package cache provides a thread-safe in-memory cache with JSON file persistence
// for POSitouch data.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"rooam-pos-agent/positouch"
)

// Data holds all cached POSitouch data.
type Data struct {
	LastRefresh time.Time             `json:"last_refresh"`
	CostCenters []positouch.CostCenter `json:"cost_centers"`
	Tenders     []positouch.Tender     `json:"tenders"`
	Employees   []positouch.Employee   `json:"employees"`
	Tables      []positouch.Table      `json:"tables"`
	OrderTypes  []positouch.OrderType  `json:"order_types"`
}

// Cache is a thread-safe in-memory cache backed by a JSON file.
type Cache struct {
	mu       sync.RWMutex
	data     Data
	filePath string
}

// New creates a new Cache that will persist to the given file path.
// If the file already exists, its contents are loaded into memory.
func New(filePath string) (*Cache, error) {
	c := &Cache{filePath: filePath}
	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("cache: load %q: %w", filePath, err)
	}
	return c, nil
}

// Update atomically replaces all cached data with the provided values and
// records the refresh timestamp.
func (c *Cache) Update(
	costCenters []positouch.CostCenter,
	tenders []positouch.Tender,
	employees []positouch.Employee,
	tables []positouch.Table,
	orderTypes []positouch.OrderType,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = Data{
		LastRefresh: time.Now().UTC(),
		CostCenters: costCenters,
		Tenders:     tenders,
		Employees:   employees,
		Tables:      tables,
		OrderTypes:  orderTypes,
	}
}

// Get returns a copy of the currently cached data.
func (c *Cache) Get() Data {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// Save persists the current in-memory cache to disk as JSON.
func (c *Cache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := os.WriteFile(c.filePath, b, 0o644); err != nil {
		return fmt.Errorf("cache: write %q: %w", c.filePath, err)
	}
	return nil
}

// load reads the cache file from disk into memory. Returns os.ErrNotExist if
// the file does not yet exist.
func (c *Cache) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c.data)
}

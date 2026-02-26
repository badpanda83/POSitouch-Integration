// Package cache provides a thread-safe in-memory cache with JSON persistence
// for the POSitouch integration agent.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"rooam-pos-agent/positouch"
)

// Cache holds all pulled POSitouch data and supports concurrent access.
type Cache struct {
	mu          sync.RWMutex
	CostCenters []positouch.CostCenter `json:"cost_centers"`
	Tenders     []positouch.Tender     `json:"tenders"`
	Employees   []positouch.Employee   `json:"employees"`
	Tables      []positouch.Table      `json:"tables"`
	OrderTypes  []positouch.OrderType  `json:"order_types"`
	LastUpdated time.Time              `json:"last_updated"`

	filePath string
}

// New creates an empty Cache that will persist to filePath.
func New(filePath string) *Cache {
	return &Cache{filePath: filePath}
}

// Update atomically replaces all cached data and records the update time.
func (c *Cache) Update(
	costCenters []positouch.CostCenter,
	tenders []positouch.Tender,
	employees []positouch.Employee,
	tables []positouch.Table,
	orderTypes []positouch.OrderType,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.CostCenters = costCenters
	c.Tenders = tenders
	c.Employees = employees
	c.Tables = tables
	c.OrderTypes = orderTypes
	c.LastUpdated = time.Now()
}

// Save writes the cache to its JSON file in a human-readable format.
func (c *Cache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshalling: %w", err)
	}
	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("cache: writing %s: %w", c.filePath, err)
	}
	return nil
}

// Load reads previously saved cache data from the JSON file.  It is safe to
// call Load on a new Cache; if the file does not exist it is silently ignored.
func (c *Cache) Load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // first run — no cache file yet
		}
		return fmt.Errorf("cache: reading %s: %w", c.filePath, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("cache: parsing %s: %w", c.filePath, err)
	}
	return nil
}

// Snapshot returns a point-in-time copy of the current cache contents.
func (c *Cache) Snapshot() (
	costCenters []positouch.CostCenter,
	tenders []positouch.Tender,
	employees []positouch.Employee,
	tables []positouch.Table,
	orderTypes []positouch.OrderType,
	lastUpdated time.Time,
) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.CostCenters, c.Tenders, c.Employees, c.Tables, c.OrderTypes, c.LastUpdated
}

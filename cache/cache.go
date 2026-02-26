// Package cache provides a thread-safe in-memory cache with JSON file
// persistence for POSitouch integration data.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// Cache holds all POSitouch data pulled by the agent.
type Cache struct {
	mu          sync.RWMutex
	CostCenters []positouch.CostCenter `json:"cost_centers"`
	Tenders     []positouch.Tender     `json:"tenders"`
	Employees   []positouch.Employee   `json:"employees"`
	Tables      []positouch.Table      `json:"tables"`
	OrderTypes  []positouch.OrderType  `json:"order_types"`
	LastUpdated time.Time              `json:"last_updated"`
}

// New returns an initialised (empty) Cache.
func New() *Cache {
	return &Cache{
		CostCenters: []positouch.CostCenter{},
		Tenders:     []positouch.Tender{},
		Employees:   []positouch.Employee{},
		Tables:      []positouch.Table{},
		OrderTypes:  []positouch.OrderType{},
	}
}

// Update replaces all cache contents atomically and stamps LastUpdated.
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

// Save serialises the cache to a JSON file at path.
func (c *Cache) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshalling: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("cache: writing %s: %w", path, err)
	}
	return nil
}

// Load reads a previously saved cache JSON file into c.
// Non-existent files are silently ignored (fresh start).
func (c *Cache) Load(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cache: reading %s: %w", path, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("cache: parsing %s: %w", path, err)
	}
	return nil
}

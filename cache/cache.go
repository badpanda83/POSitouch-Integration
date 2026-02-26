// Package cache provides an in-memory data store with JSON persistence.
// The cache holds the most recently extracted POSitouch data and writes it
// to rooam_cache.json after every refresh.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// Cache holds all extracted POSitouch data plus metadata.
type Cache struct {
	mu sync.RWMutex

	LastUpdated time.Time                `json:"last_updated"`
	Status      string                   `json:"status"`
	CostCenters []positouch.CostCenter   `json:"cost_centers"`
	Tenders     []positouch.Tender       `json:"tenders"`
	Employees   []positouch.Employee     `json:"employees"`
	Tables      []positouch.Table        `json:"tables"`
	OrderTypes  []positouch.OrderType    `json:"order_types"`
	Errors      []string                 `json:"errors"`
}

// New returns an empty cache with status "initializing".
func New() *Cache {
	return &Cache{
		Status:      "initializing",
		CostCenters: []positouch.CostCenter{},
		Tenders:     []positouch.Tender{},
		Employees:   []positouch.Employee{},
		Tables:      []positouch.Table{},
		OrderTypes:  []positouch.OrderType{},
		Errors:      []string{},
	}
}

// Update atomically replaces cache contents and persists to disk.
func (c *Cache) Update(
	costCenters []positouch.CostCenter,
	tenders []positouch.Tender,
	employees []positouch.Employee,
	tables []positouch.Table,
	orderTypes []positouch.OrderType,
	errs []string,
	cachePath string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastUpdated = time.Now()
	c.CostCenters = costCenters
	c.Tenders = tenders
	c.Employees = employees
	c.Tables = tables
	c.OrderTypes = orderTypes
	c.Errors = errs

	if len(errs) == 0 {
		c.Status = "ok"
	} else {
		c.Status = "degraded"
	}

	return c.flush(cachePath)
}

// flush writes the current cache state to disk as JSON.
// Must be called with c.mu held.
func (c *Cache) flush(cachePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling cache: %w", err)
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing cache file %s: %w", cachePath, err)
	}
	return nil
}

// Snapshot returns a copy of the current cache contents (thread-safe).
func (c *Cache) Snapshot() Cache {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Cache{
		LastUpdated: c.LastUpdated,
		Status:      c.Status,
		CostCenters: c.CostCenters,
		Tenders:     c.Tenders,
		Employees:   c.Employees,
		Tables:      c.Tables,
		OrderTypes:  c.OrderTypes,
		Errors:      c.Errors,
	}
}

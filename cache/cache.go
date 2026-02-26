// Package cache provides a thread-safe in-memory and JSON-file-backed cache
// for POSitouch integration data.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// CostCenter represents a POSitouch cost center.
type CostCenter struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// Tender represents a POSitouch payment / tender type.
type Tender struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// Employee represents a POSitouch employee / user.
type Employee struct {
	UserNumber  int    `json:"user_number"`
	LastName    string `json:"last_name"`
	FirstName   string `json:"first_name"`
	Type        int    `json:"type"`
	MagCardID   int    `json:"mag_card_id"`
	Status      string `json:"status,omitempty"`
	Phone       string `json:"phone,omitempty"`
	DateHired   string `json:"date_hired,omitempty"`
}

// Table represents a unique table/cost-center pair.
type Table struct {
	Number     int `json:"number"`
	CostCenter int `json:"cost_center"`
}

// OrderType represents a POSitouch menu / order type.
type OrderType struct {
	MenuNumber        int    `json:"menu_number"`
	Title             string `json:"title"`
	ContinuesOnMenu   int    `json:"continues_on_menu"`
	TaxCode           int    `json:"tax_code"`
	FastFoodOrderType int    `json:"fast_food_order_type"`
}

// CacheData is the full snapshot stored in the cache.
type CacheData struct {
	CostCenters []CostCenter `json:"cost_centers"`
	Tenders     []Tender     `json:"tenders"`
	Employees   []Employee   `json:"employees"`
	Tables      []Table      `json:"tables"`
	OrderTypes  []OrderType  `json:"order_types"`
	LastUpdated time.Time    `json:"last_updated"`
}

// Cache is a thread-safe in-memory store with optional JSON persistence.
type Cache struct {
	mu   sync.RWMutex
	data *CacheData
}

// New returns an empty, initialized Cache.
func New() *Cache {
	return &Cache{data: &CacheData{}}
}

// Update atomically replaces the cached data.
func (c *Cache) Update(data *CacheData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
}

// Get returns a copy of the current cache snapshot.
func (c *Cache) Get() *CacheData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a shallow copy to avoid external mutation of the stored pointer.
	cp := *c.data
	return &cp
}

// SaveToFile serialises the cache to a JSON file at path.
func (c *Cache) SaveToFile(path string) error {
	c.mu.RLock()
	data := c.data
	c.mu.RUnlock()

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("cache: write %q: %w", path, err)
	}
	return nil
}

// LoadFromFile reads a JSON file and populates the cache.
func (c *Cache) LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cache: read %q: %w", path, err)
	}
	var data CacheData
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("cache: parse %q: %w", path, err)
	}
	c.mu.Lock()
	c.data = &data
	c.mu.Unlock()
	return nil
}

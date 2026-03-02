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

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// CacheFile is the name of the on-disk cache file written to the install directory.
const CacheFile = "rooam_cache.json"

// Data holds all cached POSitouch data.
type Data struct {
	LastUpdated        time.Time                `json:"last_updated"`
	CostCenters        []positouch.CostCenter   `json:"cost_centers"`
	Tenders            []positouch.Tender       `json:"tenders"`
	Employees          []positouch.Employee     `json:"employees"`
	Tables             []positouch.Table        `json:"tables"`
	OrderTypes         []positouch.OrderType    `json:"order_types"`
	CurrentTickets     []positouch.Ticket       `json:"current_tickets"`
	HistoricalTickets  []positouch.Ticket       `json:"historical_tickets"`
	MenuItems          []positouch.MenuItem     `json:"menu_items"`
	Modifiers          []positouch.Modifier     `json:"modifiers"`
	Categories         []positouch.Category     `json:"categories"`
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

// --------------- DISK CACHE HELPERS FOR ALL ENTITIES ---------------

func WriteCostCentersToCache(costCenters []positouch.CostCenter, path string) error {
	return writeJSON(costCenters, path)
}
func WriteTendersToCache(tenders []positouch.Tender, path string) error {
	return writeJSON(tenders, path)
}
func WriteEmployeesToCache(employees []positouch.Employee, path string) error {
	return writeJSON(employees, path)
}
func WriteTablesToCache(tables []positouch.Table, path string) error {
	return writeJSON(tables, path)
}
func WriteOrderTypesToCache(orderTypes []positouch.OrderType, path string) error {
	return writeJSON(orderTypes, path)
}
func WriteTicketsToCache(tickets []positouch.Ticket, path string) error {
	return writeJSON(tickets, path)
}
func WriteMenuItemsToCache(menuItems []positouch.MenuItem, path string) error {
	return writeJSON(menuItems, path)
}
func WriteModifiersToCache(modifiers []positouch.Modifier, path string) error {
	return writeJSON(modifiers, path)
}
func WriteCategoriesToCache(categories []positouch.Category, path string) error {
	return writeJSON(categories, path)
}

func ReadCostCentersFromCache(path string) ([]positouch.CostCenter, error) {
	var v []positouch.CostCenter
	return v, readJSON(&v, path)
}
func ReadTendersFromCache(path string) ([]positouch.Tender, error) {
	var v []positouch.Tender
	return v, readJSON(&v, path)
}
func ReadEmployeesFromCache(path string) ([]positouch.Employee, error) {
	var v []positouch.Employee
	return v, readJSON(&v, path)
}
func ReadTablesFromCache(path string) ([]positouch.Table, error) {
	var v []positouch.Table
	return v, readJSON(&v, path)
}
func ReadOrderTypesFromCache(path string) ([]positouch.OrderType, error) {
	var v []positouch.OrderType
	return v, readJSON(&v, path)
}
func ReadTicketsFromCache(path string) ([]positouch.Ticket, error) {
	var v []positouch.Ticket
	return v, readJSON(&v, path)
}
func ReadMenuItemsFromCache(path string) ([]positouch.MenuItem, error) {
	var v []positouch.MenuItem
	return v, readJSON(&v, path)
}
func ReadModifiersFromCache(path string) ([]positouch.Modifier, error) {
	var v []positouch.Modifier
	return v, readJSON(&v, path)
}
func ReadCategoriesFromCache(path string) ([]positouch.Category, error) {
	var v []positouch.Category
	return v, readJSON(&v, path)
}

// --- Utility helpers for disk I/O ---

func writeJSON(v interface{}, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(v)
}
func readJSON(v interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}
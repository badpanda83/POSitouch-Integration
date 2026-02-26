package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/badpanda83/POSitouch-Integration/rooam-pos-agent/positouch"
)

func TestCache_UpdateAndGet(t *testing.T) {
	dir := t.TempDir()
	c, err := New(filepath.Join(dir, "test_cache.json"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	centers := []positouch.CostCenter{{Code: 1, Name: "Bar"}}
	tenders := []positouch.Tender{{Code: 2, Name: "Cash"}}
	employees := []positouch.Employee{{UserNumber: 42, LastName: "Smith", FirstName: "John"}}
	tables := []positouch.Table{{TableNumber: 10, CostCenter: 1}}
	orderTypes := []positouch.OrderType{{MenuNumber: 1, MenuTitle: "Dinner"}}

	c.Update(centers, tenders, employees, tables, orderTypes)

	got := c.Get()
	if len(got.CostCenters) != 1 || got.CostCenters[0].Name != "Bar" {
		t.Errorf("CostCenters = %v, want [{1 Bar}]", got.CostCenters)
	}
	if len(got.Tenders) != 1 || got.Tenders[0].Name != "Cash" {
		t.Errorf("Tenders = %v, want [{2 Cash}]", got.Tenders)
	}
	if got.LastRefresh.IsZero() {
		t.Error("LastRefresh is zero")
	}
	if time.Since(got.LastRefresh) > 5*time.Second {
		t.Errorf("LastRefresh is too old: %v", got.LastRefresh)
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_cache.json")

	c, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	centers := []positouch.CostCenter{{Code: 5, Name: "Patio"}}
	c.Update(centers, nil, nil, nil, nil)

	if err := c.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	// Load a new cache from the same file
	c2, err := New(path)
	if err != nil {
		t.Fatalf("New() load error: %v", err)
	}
	got := c2.Get()
	if len(got.CostCenters) != 1 || got.CostCenters[0].Name != "Patio" {
		t.Errorf("loaded CostCenters = %v, want [{5 Patio}]", got.CostCenters)
	}
}

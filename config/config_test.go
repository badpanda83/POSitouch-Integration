package config

import (
"os"
"path/filepath"
"strings"
"testing"
)

const sampleConfig = `{
  "location": {
    "name": "Test Bar",
    "country": "United States",
    "address1": "1 Main St",
    "city": "New York",
    "state": "NY",
    "zip": "10001"
  },
  "rooam": {
    "tender_id": "50",
    "employee_id": "9999"
  },
  "positouch": {
    "spcwin_path": "C:\\SC\\SPCWIN.ini"
  }
}`

func TestLoad(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "rooam_config_*.json")
if err != nil {
t.Fatal(err)
}
if _, err := f.WriteString(sampleConfig); err != nil {
t.Fatal(err)
}
f.Close()

cfg, err := Load(f.Name())
if err != nil {
t.Fatalf("Load error: %v", err)
}

if cfg.Location.Name != "Test Bar" {
t.Errorf("Location.Name: got %q", cfg.Location.Name)
}
if cfg.Rooam.TenderID != "50" {
t.Errorf("Rooam.TenderID: got %q", cfg.Rooam.TenderID)
}

// SCDir should end with a separator and contain "SC".
if !strings.HasSuffix(cfg.SCDir, string(filepath.Separator)) {
t.Errorf("SCDir should end with separator, got %q", cfg.SCDir)
}
if !strings.Contains(cfg.SCDir, "SC") {
t.Errorf("SCDir should contain 'SC', got %q", cfg.SCDir)
}
if !strings.Contains(cfg.DBFDir, "DBF") {
t.Errorf("DBFDir should contain 'DBF', got %q", cfg.DBFDir)
}
if !strings.Contains(cfg.ALTDBFDir, "ALTDBF") {
t.Errorf("ALTDBFDir should contain 'ALTDBF', got %q", cfg.ALTDBFDir)
}
}

func TestLoad_Missing(t *testing.T) {
_, err := Load("/nonexistent/rooam_config.json")
if err == nil {
t.Fatal("expected error for missing file, got nil")
}
}

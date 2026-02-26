package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndPaths(t *testing.T) {
	const jsonData = `{
  "location": {"name": "Test Bar"},
  "rooam": {"tender_id": "50", "employee_id": "9999"},
  "positouch": {"spcwin_path": "C:\\SC\\SPCWIN.ini"}
}`
	tmp := filepath.Join(t.TempDir(), "rooam_config.json")
	if err := os.WriteFile(tmp, []byte(jsonData), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Location.Name != "Test Bar" {
		t.Errorf("Location.Name = %q, want %q", cfg.Location.Name, "Test Bar")
	}
	if cfg.Rooam.TenderID != "50" {
		t.Errorf("Rooam.TenderID = %q, want %q", cfg.Rooam.TenderID, "50")
	}

	dbfPath := cfg.DBFPath()
	if dbfPath == "" {
		t.Fatal("DBFPath() returned empty string")
	}
	// Should start with C: and contain DBF.
	if len(dbfPath) < 2 || dbfPath[0] != 'C' {
		t.Errorf("DBFPath() = %q, want C: prefix", dbfPath)
	}

	scPath := cfg.SCPath()
	if scPath == "" {
		t.Fatal("SCPath() returned empty string")
	}
	if len(scPath) < 2 || scPath[0] != 'C' {
		t.Errorf("SCPath() = %q, want C: prefix", scPath)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestDBFPathDriveLetter(t *testing.T) {
	cfg := &AppConfig{
		POSitouch: POSitouchConfig{SPCWinPath: `D:\SC\SPCWIN.ini`},
	}
	dbfPath := cfg.DBFPath()
	if len(dbfPath) < 1 || dbfPath[0] != 'D' {
		t.Errorf("DBFPath() = %q, want D: prefix", dbfPath)
	}
	scPath := cfg.SCPath()
	if len(scPath) < 1 || scPath[0] != 'D' {
		t.Errorf("SCPath() = %q, want D: prefix", scPath)
	}
}

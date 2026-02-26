package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_DerivePaths(t *testing.T) {
	// Use a platform-native path for the SPCWIN.ini location so that
	// filepath.Dir works correctly regardless of OS.
	scDir := filepath.Join(t.TempDir(), "SC")
	if err := os.MkdirAll(scDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spcwinPath := filepath.Join(scDir, "SPCWIN.ini")

	// Write a minimal rooam_config.json to a temp file using the native path.
	content := `{
  "location": {"name": "Test Bar"},
  "rooam": {"tender_id": "50", "employee_id": "9999"},
  "positouch": {"spcwin_path": "` + strings.ReplaceAll(spcwinPath, `\`, `\\`) + `"}
}`
	tmp, err := os.CreateTemp(t.TempDir(), "rooam_config*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	wantSC := scDir + string(filepath.Separator)
	if cfg.SCPath != wantSC {
		t.Errorf("SCPath = %q, want %q", cfg.SCPath, wantSC)
	}

	if !strings.HasSuffix(cfg.DBFPath, "DBF"+string(filepath.Separator)) {
		t.Errorf("DBFPath = %q, expected to end with DBF%c", cfg.DBFPath, filepath.Separator)
	}

	if !strings.HasSuffix(cfg.ALTDBFPath, "ALTDBF"+string(filepath.Separator)) {
		t.Errorf("ALTDBFPath = %q, expected to end with ALTDBF%c", cfg.ALTDBFPath, filepath.Separator)
	}
}

func TestLoad_MissingSpcwinPath(t *testing.T) {
	content := `{"location": {}, "rooam": {}, "positouch": {"spcwin_path": ""}}`
	tmp, err := os.CreateTemp(t.TempDir(), "rooam_config*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	_, err = Load(tmp.Name())
	if err == nil {
		t.Error("expected error for empty spcwin_path, got nil")
	}
}

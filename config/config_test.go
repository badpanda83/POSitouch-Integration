package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/badpanda83/POSitouch-Integration/config"
)

func writeConfig(t *testing.T, dir string, cfg map[string]interface{}) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestLoad_DerivesDBFPath(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"positouch": map[string]interface{}{
			"spcwin_path": `C:\SC\SPCWIN.ini`,
		},
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBFDir != `C:\DBF` {
		t.Errorf("DBFDir: want %q, got %q", `C:\DBF`, cfg.DBFDir)
	}
	if cfg.AltDBFDir != `C:\ALTDBF` {
		t.Errorf("AltDBFDir: want %q, got %q", `C:\ALTDBF`, cfg.AltDBFDir)
	}
}

func TestLoad_EmptySpcwinPath(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"positouch": map[string]interface{}{
			"spcwin_path": "",
		},
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBFDir != `C:\DBF` {
		t.Errorf("DBFDir: want %q, got %q", `C:\DBF`, cfg.DBFDir)
	}
}

func TestLoad_DifferentDriveLetter(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"positouch": map[string]interface{}{
			"spcwin_path": `D:\SC\SPCWIN.ini`,
		},
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBFDir != `D:\DBF` {
		t.Errorf("DBFDir: want %q, got %q", `D:\DBF`, cfg.DBFDir)
	}
	if cfg.AltDBFDir != `D:\ALTDBF` {
		t.Errorf("AltDBFDir: want %q, got %q", `D:\ALTDBF`, cfg.AltDBFDir)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestLoad_InstallDirSet(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, map[string]interface{}{
		"location": map[string]interface{}{"name": "Test Venue"},
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.InstallDir != dir {
		t.Errorf("InstallDir: want %q, got %q", dir, cfg.InstallDir)
	}
}

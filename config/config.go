// Package config reads the rooam_config.json file produced by the Inno Setup
// installer and exposes the parsed values together with derived directory paths.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// nativeSep is the OS-specific directory separator as a string.
const nativeSep = string(filepath.Separator)

// Location holds venue contact information.
type Location struct {
	Name     string `json:"name"`
	Country  string `json:"country"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// Rooam holds Rooam-specific integration settings.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds POSitouch-specific settings from the config file.
type POSitouch struct {
	SPCWinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level configuration struct.  In addition to the parsed JSON
// fields it exposes derived directory paths.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived paths (not stored in JSON).
	SCDir     string // e.g. C:\SC\
	DBFDir    string // e.g. C:\DBF\
	ALTDBFDir string // e.g. C:\ALTDBF\
}

// Load reads the JSON file at path and returns a populated Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	// Derive SC directory from the spcwin_path (just the directory part).
	// Normalise to forward slashes so the logic works cross-platform; the
	// resulting paths will still use the native separator via normSep().
	spcwinNorm := normPath(cfg.POSitouch.SPCWinPath)
	lastSep := strings.LastIndexAny(spcwinNorm, "/\\")
	if lastSep >= 0 {
		cfg.SCDir = spcwinNorm[:lastSep+1]
	} else {
		cfg.SCDir = "." + nativeSep
	}

	// Derive DBF and ALTDBF by replacing the last directory component.
	// E.g.  C:/SC/  →  C:/DBF/  and  C:/ALTDBF/
	scTrimmed := strings.TrimRight(cfg.SCDir, "/\\")
	lastSep2 := strings.LastIndexAny(scTrimmed, "/\\")
	var parent string
	if lastSep2 >= 0 {
		parent = scTrimmed[:lastSep2+1]
	} else {
		parent = ""
	}
	cfg.DBFDir = parent + "DBF" + nativeSep
	cfg.ALTDBFDir = parent + "ALTDBF" + nativeSep

	return &cfg, nil
}

// normPath replaces all backslashes with forward slashes so path manipulation
// works correctly on non-Windows hosts.
func normPath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

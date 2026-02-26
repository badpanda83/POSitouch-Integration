// Package config reads the Rooam configuration file (rooam_config.json) and
// derives filesystem paths needed by the POSitouch integration agent.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Location holds the physical location details from the config file.
type Location struct {
	Name     string `json:"name"`
	Country  string `json:"country"`
	Address1 string `json:"address1"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// Rooam holds Rooam-specific identifiers from the config file.
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

// Config is the top-level structure for rooam_config.json.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived paths (not in JSON).
	InstallDir string `json:"-"`
	DBFDir     string `json:"-"`
	AltDBFDir  string `json:"-"`
}

// DefaultInstallDir is the Rooam install directory on Windows.
const DefaultInstallDir = `C:\Program Files\Rooam\POSitouch`

// ConfigFileName is the name of the Rooam config file.
const ConfigFileName = "rooam_config.json"

// Load reads rooam_config.json from the given install directory and populates
// the derived DBF path fields.
func Load(installDir string) (*Config, error) {
	path := filepath.Join(installDir, ConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	cfg.InstallDir = installDir
	cfg.DBFDir, cfg.AltDBFDir = derivePaths(cfg.POSitouch.SPCWinPath)
	return &cfg, nil
}

// derivePaths extracts the drive letter from spcwin_path and constructs the
// DBF and ALTDBF directory paths.
//
// Example: spcwin_path = "C:\SC\SPCWIN.ini"  →  DBFDir = "C:\DBF", AltDBFDir = "C:\ALTDBF"
func derivePaths(spcwinPath string) (dbfDir, altDBFDir string) {
	if spcwinPath == "" {
		return `C:\DBF`, `C:\ALTDBF`
	}
	// Normalise separators so the code works on Linux too (useful for tests).
	normalised := strings.ReplaceAll(spcwinPath, `/`, `\`)
	// Extract drive letter: first two characters when they look like "C:".
	drive := ""
	if len(normalised) >= 2 && normalised[1] == ':' {
		drive = string(normalised[0]) + `:`
	}
	if drive == "" {
		return `C:\DBF`, `C:\ALTDBF`
	}
	return drive + `\DBF`, drive + `\ALTDBF`
}

// Package config reads the rooam_config.json file produced by the Rooam installer.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Location holds venue location metadata from the config file.
type Location struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
}

// Rooam holds Rooam-specific settings.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds POSitouch-specific settings.
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level structure of rooam_config.json.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived paths — populated by Load, not read from JSON.
	InstallDir string `json:"-"`
	SCDir      string `json:"-"`
	DBFDir     string `json:"-"`
	AltDBFDir  string `json:"-"`
}

// DefaultConfigPath is the default location of rooam_config.json on Windows.
const DefaultConfigPath = `C:\Program Files\Rooam\POSitouch\rooam_config.json`

// Load reads and parses rooam_config.json from the given file path and derives
// the POSitouch directory paths from the spcwin_path field.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", configPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", configPath, err)
	}

	cfg.InstallDir = filepath.Dir(configPath)
	cfg.derivePaths()
	return &cfg, nil
}

// derivePaths sets SCDir, DBFDir, and AltDBFDir from the spcwin_path field.
// e.g. spcwin_path = "C:\SC\SPCWIN.ini"  →  drive = "C:"
//
//	SCDir    = "C:\SC"
//	DBFDir   = "C:\DBF"
//	AltDBFDir= "C:\ALTDBF"
func (c *Config) derivePaths() {
	if c.POSitouch.SpcwinPath == "" {
		return
	}
	c.SCDir = filepath.Dir(c.POSitouch.SpcwinPath)
	drive := filepath.VolumeName(c.SCDir)
	c.DBFDir = filepath.Join(drive+string(filepath.Separator), "DBF")
	c.AltDBFDir = filepath.Join(drive+string(filepath.Separator), "ALTDBF")
}

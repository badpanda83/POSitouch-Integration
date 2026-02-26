// Package config loads and exposes the Rooam POSitouch configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultConfigPath is the default location of the Rooam config file on Windows.
const DefaultConfigPath = `C:\Program Files\Rooam\POSitouch\rooam_config.json`

// LocationConfig holds the restaurant location details.
type LocationConfig struct {
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

// RooamConfig holds Rooam-specific identifiers.
type RooamConfig struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouchConfig holds POSitouch-specific settings.
type POSitouchConfig struct {
	SPCWinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// AppConfig is the top-level configuration structure.
type AppConfig struct {
	Location  LocationConfig  `json:"location"`
	Rooam     RooamConfig     `json:"rooam"`
	POSitouch POSitouchConfig `json:"positouch"`
}

// Load reads and parses the JSON config file at the given path.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}
	return &cfg, nil
}

// DBFPath derives the DBF directory from the spcwin_path in the config.
// The DBF files live in \DBF\ on the same drive as \SC\.
// e.g. spcwin_path = "C:\SC\SPCWIN.ini" → "C:\DBF\"
func (c *AppConfig) DBFPath() string {
	drive := driveLetter(c.POSitouch.SPCWinPath)
	return filepath.Join(drive+`\`, "DBF") + string(filepath.Separator)
}

// SCPath returns the \SC\ directory path derived from spcwin_path.
// e.g. spcwin_path = "C:\SC\SPCWIN.ini" → "C:\SC\"
func (c *AppConfig) SCPath() string {
	drive := driveLetter(c.POSitouch.SPCWinPath)
	return filepath.Join(drive+`\`, "SC") + string(filepath.Separator)
}

// driveLetter extracts the drive letter (e.g. "C") from a Windows path.
// Falls back to "C" if no drive letter can be determined.
func driveLetter(path string) string {
	if len(path) >= 2 && path[1] == ':' {
		return strings.ToUpper(string(path[0]))
	}
	return "C"
}

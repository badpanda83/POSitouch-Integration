// Package config reads the rooam_config.json file written by the Inno Setup installer
// and derives the POSitouch DBF directory paths.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultConfigPath is the default location of the config file on Windows.
const DefaultConfigPath = `C:\Program Files\Rooam\POSitouch\rooam_config.json`

// Location holds venue contact and address information.
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

// Rooam holds Rooam-specific integration IDs.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds POSitouch-specific paths and configuration.
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level configuration object.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// configDir is the directory that contains the config file itself.
	configDir string
}

// Load reads and parses the config file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}
	cfg.configDir = filepath.Dir(path)
	return &cfg, nil
}

// ConfigDir returns the directory that contains the config file (i.e. the install directory).
func (c *Config) ConfigDir() string {
	return c.configDir
}

// scDir derives the SC directory from spcwin_path.
// e.g. "C:\SC\SPCWIN.ini" → "C:\SC\"
func (c *Config) SCDir() string {
	p := c.POSitouch.SpcwinPath
	if p == "" {
		return ""
	}
	// normalise to forward-slash for path manipulation, then convert back
	p = filepath.FromSlash(strings.ReplaceAll(p, `\`, `/`))
	return filepath.Dir(p) + string(filepath.Separator)
}

// DBFDir returns the DBF directory — always a sibling of the SC folder at the drive root.
// e.g. "C:\SC\" → "C:\DBF\"
func (c *Config) DBFDir() string {
	sc := c.SCDir()
	if sc == "" {
		return ""
	}
	vol := filepath.VolumeName(sc)
	return vol + string(filepath.Separator) + "DBF" + string(filepath.Separator)
}

// AltDBFDir returns the alternate DBF directory.
// e.g. "C:\SC\" → "C:\ALTDBF\"
func (c *Config) AltDBFDir() string {
	sc := c.SCDir()
	if sc == "" {
		return ""
	}
	vol := filepath.VolumeName(sc)
	return vol + string(filepath.Separator) + "ALTDBF" + string(filepath.Separator)
}

package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)
const DefaultConfigPath = "rooam_config.json"
// ----- Location definition -----
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

// ----- Rooam-specific fields -----
type Rooam struct {
    TenderID   string `json:"tender_id"`
    EmployeeID string `json:"employee_id"`
}

// ----- POSitouch fields -----
type POSitouch struct {
    SpcwinPath     string `json:"spcwin_path"`
    VirtualSection string `json:"virtual_section"`
    XMLSection     string `json:"xml_section"`
}

// ----- Cloud config -----
type CloudConfig struct {
    Enabled  bool   `json:"enabled"`
    Endpoint string `json:"endpoint"`
    APIKey   string `json:"api_key"`
}

// ----- Top-level Config -----
type Config struct {
    Location    Location    `json:"location"`
    Rooam       Rooam       `json:"rooam"`
    POSitouch   POSitouch   `json:"positouch"`
    Cloud       CloudConfig `json:"cloud"`

    XMLDir         string      `json:"xml_dir"`           // open tickets directory
    XMLCloseDir    string      `json:"xml_close_dir"`     // closed tickets directory
    XMLInOrderDir  string      `json:"xml_inorder_dir"`   // inbound order drop directory

    SCDir      string
    SCPath     string
    DBFDir     string
    DBFPath    string
    ALTDBFDir  string
    ALTDBFPath string
    AltDBFDir  string    // (CamelCase for main.go compatibility)
    InstallDir string    // Directory containing config file
}

// Load reads the config JSON file and computes the paths used by the agent.
func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("config: read %q: %w", path, err)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("config: parse %q: %w", path, err)
    }

    if cfg.POSitouch.SpcwinPath == "" {
        return nil, fmt.Errorf("config: positouch.spcwin_path is empty")
    }

    scDir := filepath.Dir(cfg.POSitouch.SpcwinPath)
    parentDir := filepath.Dir(scDir)
    dbfDir := filepath.Join(parentDir, "DBF")
    altdbfDir := filepath.Join(parentDir, "ALTDBF")

    cfg.SCPath     = scDir + string(filepath.Separator)
    cfg.SCDir      = cfg.SCPath
    cfg.DBFPath    = dbfDir + string(filepath.Separator)
    cfg.DBFDir     = cfg.DBFPath
    cfg.ALTDBFPath = altdbfDir + string(filepath.Separator)
    cfg.ALTDBFDir  = cfg.ALTDBFPath
    cfg.AltDBFDir  = cfg.ALTDBFDir

    cfg.InstallDir = filepath.Dir(path)

    return &cfg, nil
}
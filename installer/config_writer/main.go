// config_writer writes a rooam_config.json file for the Rooam POS Agent.
//
// It is a small, standalone helper that is invoked by the WiX custom-action
// DLL (or by the PowerShell build script during development) to merge
// user-supplied installation parameters with the static defaults and write a
// valid configuration file to disk.
//
// This program is intentionally NOT part of the root module. It is compiled
// separately so that it can be embedded in the MSI without dragging in the
// agent's full dependency tree.
//
// Usage:
//
//	config_writer.exe [flags]
//
// Flags:
//
//	-location-name    string  Venue / restaurant name
//	-location-id      string  Location identifier (defaults to location-name if blank)
//	-address          string  Street address (address1)
//	-phone            string  Contact phone number
//	-email            string  Contact e-mail address
//	-employee-id      string  Rooam employee identifier
//	-tender-id        string  Rooam tender identifier
//	-api-key          string  Cloud API key (Bearer token)
//	-spcwin-path      string  Full path to spcwin.exe
//	-xml-dir          string  Open-tickets XML directory
//	-xml-close-dir    string  Closed-tickets XML directory
//	-xml-inorder-dir  string  Inbound-order XML directory
//	-output           string  Destination file path (default: same dir as exe)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Static defaults — keep in sync with the production cloud server URL.
const (
	defaultCloudEndpoint = "https://positouch-cloud-server-production.up.railway.app/api/v1/pos-data"
	defaultPOSType       = "positouch"
	defaultConfigName    = "rooam_config.json"
)

// location mirrors config.Location (copied here to avoid a module dependency).
type location struct {
	Name    string `json:"name"`
	Country string `json:"country,omitempty"`
	Address string `json:"address1,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
}

// rooam mirrors config.Rooam.
type rooam struct {
	TenderID   string `json:"tender_id,omitempty"`
	EmployeeID string `json:"employee_id,omitempty"`
}

// positouch mirrors config.POSitouch.
type positouch struct {
	SpcwinPath string `json:"spcwin_path"`
}

// cloudConfig mirrors config.CloudConfig.
type cloudConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key,omitempty"`
}

// rooamConfig is the full rooam_config.json schema (POSitouch variant).
type rooamConfig struct {
	Location      location    `json:"location"`
	Rooam         rooam       `json:"rooam"`
	POSitouch     positouch   `json:"positouch"`
	Cloud         cloudConfig `json:"cloud"`
	POSType       string      `json:"pos_type"`
	LocationID    string      `json:"location_id,omitempty"`
	XMLDir        string      `json:"xml_dir"`
	XMLCloseDir   string      `json:"xml_close_dir"`
	XMLInOrderDir string      `json:"xml_inorder_dir"`
}

func main() {
	// ------------------------------------------------------------------ flags
	locationName  := flag.String("location-name", "", "Venue / restaurant name (required)")
	locationID    := flag.String("location-id", "", "Location identifier (defaults to location-name if blank)")
	address       := flag.String("address", "", "Street address (address1)")
	phone         := flag.String("phone", "", "Contact phone number")
	email         := flag.String("email", "", "Contact e-mail address")
	employeeID    := flag.String("employee-id", "", "Rooam employee identifier")
	tenderID      := flag.String("tender-id", "", "Rooam tender identifier")
	apiKey        := flag.String("api-key", "", "Cloud API key")
	spcwinPath    := flag.String("spcwin-path", `C:\SC\spcwin.exe`, "Full path to spcwin.exe")
	xmlDir        := flag.String("xml-dir", `C:\SC\XML`, "Open-tickets XML directory")
	xmlCloseDir   := flag.String("xml-close-dir", `C:\SC\XMLCLOSE`, "Closed-tickets XML directory")
	xmlInOrderDir := flag.String("xml-inorder-dir", `C:\SC\INORDER`, "Inbound-order XML directory")
	outputPath    := flag.String("output", "", "Destination file path (default: <exe dir>/rooam_config.json)")

	flag.Parse()

	// ------------------------------------------------------------------ resolve output path
	if *outputPath == "" {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config_writer: could not determine executable path: %v\n", err)
			os.Exit(1)
		}
		*outputPath = filepath.Join(filepath.Dir(exe), defaultConfigName)
	}

	// ------------------------------------------------------------------ build config
	resolvedLocationID := *locationID
	if resolvedLocationID == "" {
		resolvedLocationID = *locationName
	}

	cfg := rooamConfig{
		Location: location{
			Name:    *locationName,
			Country: "US",
			Address: *address,
			Phone:   *phone,
			Email:   *email,
		},
		Rooam: rooam{
			TenderID:   *tenderID,
			EmployeeID: *employeeID,
		},
		POSitouch: positouch{
			SpcwinPath: *spcwinPath,
		},
		Cloud: cloudConfig{
			Enabled:  true,
			Endpoint: defaultCloudEndpoint,
			APIKey:   *apiKey,
		},
		POSType:       defaultPOSType,
		LocationID:    resolvedLocationID,
		XMLDir:        *xmlDir,
		XMLCloseDir:   *xmlCloseDir,
		XMLInOrderDir: *xmlInOrderDir,
	}

	// ------------------------------------------------------------------ ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "config_writer: create directory: %v\n", err)
		os.Exit(1)
	}

	// ------------------------------------------------------------------ write
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config_writer: marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "config_writer: write %s: %v\n", *outputPath, err)
		os.Exit(1)
	}

	fmt.Printf("config_writer: wrote %s\n", *outputPath)
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/config"
)

// prompt prints "label [defaultVal]: " and reads a line from the scanner.
// If the user enters nothing, defaultVal is returned.
func prompt(scanner *bufio.Scanner, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return defaultVal
}

// runWizard runs an interactive terminal prompt sequence and writes
// rooam_config.json in the current working directory.
func runWizard() (*config.Config, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("=== POS Agent Configuration Wizard ===")
	fmt.Println()

	// POS type selection
	fmt.Println("POS System type:")
	fmt.Println("  1. POSitouch")
	fmt.Println("  2. MICROS 3700")
	posChoice := prompt(scanner, "Enter choice", "1")
	posType := "positouch"
	if posChoice == "2" {
		posType = "micros3700"
	}

	fmt.Println()
	locationName := prompt(scanner, "Location name (used as identifier with cloud server)", "")
	locationID := prompt(scanner, "Location ID (leave blank to use location name)", "")
	if locationID == "" {
		locationID = locationName
	}

	fmt.Println()
	cloudEndpoint := prompt(scanner, "Cloud server endpoint (e.g. https://your-server.railway.app/api/v1/pos-data)", "")
	cloudAPIKey := prompt(scanner, "Cloud API key", "")

	cfg := &config.Config{
		Location: config.Location{
			Name: locationName,
		},
		POSType: posType,
		Cloud: config.CloudConfig{
			Enabled:  true,
			Endpoint: cloudEndpoint,
			APIKey:   cloudAPIKey,
		},
		LocationID: locationID,
	}

	if posType == "positouch" {
		fmt.Println()
		fmt.Println("--- POSitouch settings ---")
		spcwinPath := prompt(scanner, `SPCWIN.EXE path (e.g. C:\SC\spcwin.exe)`, `C:\SC\spcwin.exe`)
		xmlDir := prompt(scanner, `XML open tickets directory (xml_dir)`, `C:\SC\XML`)
		xmlCloseDir := prompt(scanner, `XML closed tickets directory (xml_close_dir)`, `C:\SC\XML\CLOSE`)
		xmlInOrderDir := prompt(scanner, `XML inbound order directory (xml_inorder_dir)`, `C:\SC\XMLIN`)

		cfg.POSitouch = config.POSitouch{
			SpcwinPath: spcwinPath,
		}
		cfg.XMLDir = xmlDir
		cfg.XMLCloseDir = xmlCloseDir
		cfg.XMLInOrderDir = xmlInOrderDir
	} else {
		fmt.Println()
		fmt.Println("--- MICROS 3700 settings ---")
		tsURL := prompt(scanner, "Transaction Services URL (e.g. http://micros-server:8008/TransactionServices)", "")
		dbHost := prompt(scanner, "Database host", "")
		dbName := prompt(scanner, "Database name", "")
		dbUser := prompt(scanner, "Database user", "")
		dbPass := prompt(scanner, "Database password", "")

		rcIDStr := prompt(scanner, "Revenue Center ID (0 = not set)", "0")
		rcID, err := strconv.Atoi(rcIDStr)
		if err != nil {
			rcID = 0
		}
		termIDStr := prompt(scanner, "Terminal ID (0 = not set)", "0")
		termID, err := strconv.Atoi(termIDStr)
		if err != nil {
			termID = 0
		}

		cfg.MICROS3700 = &config.MICROS3700Config{
			TransactionServicesURL: tsURL,
			DatabaseHost:           dbHost,
			DatabaseName:           dbName,
			DatabaseUser:           dbUser,
			DatabasePassword:       dbPass,
			RevenueCenterID:        rcID,
			TerminalID:             termID,
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("wizard: marshal config: %w", err)
	}

	if err := os.WriteFile(config.DefaultConfigPath, data, 0600); err != nil {
		return nil, fmt.Errorf("wizard: write config: %w", err)
	}

	fmt.Printf("[wizard] Config written to %s\n", config.DefaultConfigPath)
	return cfg, nil
}

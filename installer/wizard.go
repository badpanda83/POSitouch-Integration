package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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

// readIniValue opens a simple INI file and returns the value for the given key.
// Returns "" if the file cannot be read or the key is not found.
func readIniValue(iniPath, key string) string {
	f, err := os.Open(iniPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lowerKey := strings.ToLower(key)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// Match "Key=value" or "Key =value" — ensure we match the exact key name
		// followed by optional whitespace and then '='.
		rest := strings.TrimPrefix(strings.ToLower(line), lowerKey)
		if rest == strings.ToLower(line) {
			// prefix not stripped — key doesn't match
			continue
		}
		if !strings.HasPrefix(strings.TrimLeft(rest, " \t"), "=") {
			// next non-space character is not '=' — not an exact key match
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			if val := strings.TrimSpace(parts[1]); val != "" {
				return val
			}
		}
	}
	return ""
}

// detectSpcwinPath reads C:\SC\SPCWIN.ini (case-insensitive on the key) and
// returns the SpcwinPath value. Falls back to C:\SC\spcwin.exe if the file
// does not exist or the key is absent.
func detectSpcwinPath() string {
	const iniPath = `C:\SC\SPCWIN.ini`
	const defaultPath = `C:\SC\spcwin.exe`

	if val := readIniValue(iniPath, "SpcwinPath"); val != "" {
		return val
	}
	return defaultPath
}

// parseHostFromConnStr extracts the hostname from a DSN such as
// "sybase+pyodbc://user:pass@Hostname/dbname".
func parseHostFromConnStr(connStr string) string {
	if u, err := url.Parse(connStr); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	// Fallback: find the part after the last '@'.
	if idx := strings.LastIndex(connStr, "@"); idx >= 0 {
		host := connStr[idx+1:]
		if slashIdx := strings.Index(host, "/"); slashIdx >= 0 {
			host = host[:slashIdx]
		}
		if host = strings.TrimSpace(host); host != "" {
			return host
		}
	}
	return ""
}

// tcpReachable returns true if host:port is reachable within 5 seconds.
func tcpReachable(host, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// microsEmployee holds the fields we care about from a SOAP QueryEmployees response.
type microsEmployee struct {
	ObjectNum int    `xml:"EmployeeObjectNum"`
	LastName  string `xml:"OperatorLastName"`
	FirstName string `xml:"OperatorFirstName"`
}

const queryEmployeesByNameBody = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <QueryEmployees xmlns="http://www.micros.com/res/pos/webservices/general/v1">
      <request>
        <OperatorLastName>Rooam</OperatorLastName>
      </request>
    </QueryEmployees>
  </soap:Body>
</soap:Envelope>`

// querySoapRooamEmployees POSTs a SOAP QueryEmployees request to the MICROS 3700
// Transaction Services endpoint and returns employees whose last name contains
// "rooam" (case-insensitive).
//
// TODO: verify the exact XML element paths against the production WSDL —
// the response path "Body>QueryEmployeesResponse>result>Employee" is an
// approximation; adjust if the MICROS WSDL uses different wrapper names.
func querySoapRooamEmployees(soapURL string) ([]microsEmployee, error) {
	// The service endpoint is the .asmx URL without the ?wsdl query string.
	serviceURL := strings.TrimSuffix(soapURL, "?wsdl")

	req, err := http.NewRequest(http.MethodPost, serviceURL, bytes.NewBufferString(queryEmployeesByNameBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the SOAP envelope. We use a flexible nested struct so that minor
	// namespace or wrapper-name differences don't cause a hard failure.
	type employeeList struct {
		Employees []microsEmployee `xml:"Employee"`
	}
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Inner []employeeList `xml:",any"`
		} `xml:"Body"`
	}
	if xmlErr := xml.Unmarshal(data, &envelope); xmlErr != nil {
		return nil, xmlErr
	}

	var all []microsEmployee
	for _, bl := range envelope.Body.Inner {
		for _, emp := range bl.Employees {
			if strings.Contains(strings.ToLower(emp.LastName), "rooam") {
				all = append(all, emp)
			}
		}
	}
	return all, nil
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

		// Detect spcwin.exe path from C:\SC\SPCWIN.ini (SpcwinPath key).
		detectedPath := detectSpcwinPath()
		spcwinPath := prompt(scanner, `SPCWIN.EXE path`, detectedPath)

		if _, err := os.Stat(spcwinPath); err != nil {
			fmt.Printf("⚠ spcwin.exe not found at %q — continuing anyway\n", spcwinPath)
		}

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

		// Step 1: SQL database connection string.
		fmt.Println()
		connStr := prompt(scanner, "MICROS 3700 database connection string",
			"sybase+pyodbc://custom:custom@Micros")

		dbHost := parseHostFromConnStr(connStr)
		if dbHost != "" {
			if tcpReachable(dbHost, "2638") {
				fmt.Println("✓ database host reachable")
			} else {
				fmt.Println("⚠ could not reach database host — continuing anyway")
			}
		}

		// Step 2: SOAP / Transaction Services URL.
		fmt.Println()
		soapURL := prompt(scanner, "MICROS 3700 SOAP URL",
			"http://localhost/ResPosApiWeb/ResPosApiWeb.asmx?wsdl")

		soapReachable := false
		{
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(soapURL)
			if err != nil {
				fmt.Println("⚠ could not reach SOAP endpoint — continuing anyway")
			} else {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					fmt.Printf("✓ SOAP endpoint reachable: HTTP %d\n", resp.StatusCode)
					soapReachable = true
				} else {
					fmt.Printf("⚠ SOAP endpoint returned HTTP %d — continuing anyway\n", resp.StatusCode)
				}
			}
		}

		// Step 3: Find or create the "Rooam" employee.
		fmt.Println()
		employeeID := ""
		if !soapReachable {
			fmt.Println("(SOAP endpoint unreachable — skipping employee lookup)")
			employeeID = prompt(scanner, "Rooam Employee ID", "")
		} else {
			employees, err := querySoapRooamEmployees(soapURL)
			if err != nil || len(employees) == 0 {
				fmt.Println("⚠ No employee named \"Rooam\" found in MICROS.")
				fmt.Println("  Please create an employee named \"Rooam\" in MICROS Back Office, then enter the employee ID here.")
				employeeID = prompt(scanner, "Rooam Employee ID", "")
				// TODO: validate employeeID via a SOAP GetEmployee call once
				// the exact MICROS ResPosApiWeb operation is known.
			} else {
				emp := employees[0]
				name := strings.TrimSpace(emp.FirstName + " " + emp.LastName)
				if name == "" {
					name = emp.LastName
				}
				fmt.Printf("✓ Found Rooam employee: %q (ID: %d)\n", name, emp.ObjectNum)
				useThis := prompt(scanner, "Use this employee ID? [Y/n]", "Y")
				if strings.ToUpper(useThis) == "Y" || useThis == "" {
					employeeID = strconv.Itoa(emp.ObjectNum)
				} else {
					employeeID = prompt(scanner, "Rooam Employee ID", "")
				}
			}
		}

		// Step 4: Rooam service charge tender.
		fmt.Println()
		tenderID := prompt(scanner, "Rooam service charge tender ID (the tender used for Rooam payments)", "")

		cfg.Rooam = config.Rooam{
			EmployeeID: employeeID,
			TenderID:   tenderID,
		}

		cfg.MICROS3700 = &config.MICROS3700Config{
			TransactionServicesURL: soapURL,
			ConnectionString:       connStr,
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

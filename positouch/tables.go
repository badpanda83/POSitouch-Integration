// tables.go reads restaurant tables and their revenue center association from SC/set1.xml.
package positouch

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Table represents a single table and its revenue center association.
type Table struct {
	ID               int    `json:"id"`
	RevenueCenter    string `json:"revenue_center"`
	RevenueCenterID  int    `json:"revenue_center_id"`
}

// ParseTablesFromSet1XML parses SC/set1.xml for tables and their revenue centers.
func ParseTablesFromSet1XML(path string) ([]Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var doc struct {
		FieldsAndFlags struct {
			TableItems         struct {
				Items []struct {
					XMLName xml.Name
					Value   string `xml:",chardata"`
				} `xml:",any"`
			} `xml:"RestaurantLayoutTables"`
			CostCenterRawNames string `xml:"RestaurantLayoutCostCenters"`
		} `xml:"FieldsAndFlags"`
	}
	dec := xml.NewDecoder(f)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode xml: %w", err)
	}

	// Parse revenue center names (cost centers)
	centers := []string{}
	ccFields := strings.Split(doc.FieldsAndFlags.CostCenterRawNames, ",")
	for i := 0; i < len(ccFields); i += 6 { // Each cost center appears to be 6 CSV fields
		name := strings.Trim(ccFields[i], "\" ")
		if name != "" {
			centers = append(centers, name)
		}
	}

	// Parse tables
	var tables []Table
	for _, item := range doc.FieldsAndFlags.TableItems.Items {
		vals := strings.Split(item.Value, ",")
		if len(vals) < 2 {
			continue
		}
		tableNum, err1 := strconv.Atoi(strings.TrimSpace(vals[0]))
		revCenterIdx, err2 := strconv.Atoi(strings.TrimSpace(vals[1]))
		if err1 != nil || err2 != nil || tableNum == 0 {
			continue
		}
		// POSitouch uses 1-based index for rev centers, our list is 0-based, so subtract 1.
		idx := revCenterIdx - 1
		revName := ""
		if idx >= 0 && idx < len(centers) {
			revName = centers[idx]
		}
		tables = append(tables, Table{
			ID:              tableNum,
			RevenueCenter:   revName,
			RevenueCenterID: revCenterIdx,
		})
	}
	return tables, nil
}
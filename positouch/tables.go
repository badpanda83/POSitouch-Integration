package positouch

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"rooam-pos-agent/config"
	"rooam-pos-agent/dbf"
)

// Table represents a unique table with its associated cost center.
type Table struct {
	Number     int `json:"number"`
	CostCenter int `json:"cost_center"`
}

// ReadTables reads table data from CHKHDR.DBF.  If that file is not present it
// falls back to CHECK.DBF.  Both DBF and ALTDBF directories are tried.
// Duplicate table numbers are deduplicated; the first occurrence wins.
func ReadTables(cfg *config.Config) ([]Table, error) {
	// Try CHKHDR.DBF first.
	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "CHKHDR.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseTablesFromChkhdr(records), nil
		}
		log.Printf("tables: CHKHDR.DBF not found in %s, trying fallback", dir)
	}

	// Fallback: CHECK.DBF
	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "CHECK.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseTablesFromCheck(records), nil
		}
		log.Printf("tables: CHECK.DBF not found in %s", dir)
	}

	return nil, fmt.Errorf("tables: no suitable DBF file found")
}

func parseTablesFromChkhdr(records []map[string]string) []Table {
	seen := make(map[int]bool)
	out := make([]Table, 0)
	for _, r := range records {
		num, err := strconv.Atoi(strings.TrimSpace(r["TABLE"]))
		if err != nil || num == 0 {
			continue
		}
		if seen[num] {
			continue
		}
		seen[num] = true
		cc, _ := strconv.Atoi(strings.TrimSpace(r["COST_CENTR"]))
		out = append(out, Table{Number: num, CostCenter: cc})
	}
	return out
}

func parseTablesFromCheck(records []map[string]string) []Table {
	seen := make(map[int]bool)
	out := make([]Table, 0)
	for _, r := range records {
		num, err := strconv.Atoi(strings.TrimSpace(r["TABLE_NO"]))
		if err != nil || num == 0 {
			continue
		}
		if seen[num] {
			continue
		}
		seen[num] = true
		cc, _ := strconv.Atoi(strings.TrimSpace(r["COSTCENTER"]))
		out = append(out, Table{Number: num, CostCenter: cc})
	}
	return out
}

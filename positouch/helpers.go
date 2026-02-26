// Package positouch provides typed readers for POSitouch DBF data files.
// helpers.go contains shared field-extraction utilities used across all readers.
package positouch

import (
	"log"
	"strconv"
	"strings"
)

// floatField safely extracts a float64 value from a DBF record map.
func floatField(rec map[string]interface{}, key string) float64 {
	v, ok := rec[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// stringField safely extracts a string value from a DBF record map.
func stringField(rec map[string]interface{}, key string) string {
	v, ok := rec[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// parseCodeSuffix converts the numeric suffix of a NAMES.DBF code string
// (e.g. "CC001" → 1, "PY02" → 2) to an integer.
// Non-numeric suffixes are logged as a warning and return 0.
func parseCodeSuffix(suffix string) int {
	trimmed := strings.TrimLeft(suffix, "0")
	if trimmed == "" {
		return 0
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		log.Printf("[positouch] parseCodeSuffix: non-numeric suffix %q: %v", suffix, err)
		return 0
	}
	return n
}

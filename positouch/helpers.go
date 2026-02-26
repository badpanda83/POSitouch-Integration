// Shared helper utilities for POSitouch DBF record parsing.
package positouch

import (
	"strconv"
	"strings"
)

// floatField safely retrieves a float64 value from a record map.
func floatField(r map[string]interface{}, key string) float64 {
	v, ok := r[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	}
	return 0
}

// stringField safely retrieves a string value from a record map.
func stringField(r map[string]interface{}, key string) string {
	v, ok := r[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// parseCodeSuffix strips the given prefix length from a code string and
// parses the remainder as an integer (e.g. "CC03" → 3).
func parseCodeSuffix(code string, prefixLen int) int {
	if len(code) <= prefixLen {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(code[prefixLen:]))
	if err != nil {
		return 0
	}
	return n
}

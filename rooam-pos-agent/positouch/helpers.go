// helpers.go provides shared field-extraction helpers for the positouch package.
package positouch

// floatField safely extracts a float64 value from a record map.
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

// stringField safely extracts a string value from a record map.
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

// boolField safely extracts a bool value from a record map.
func boolField(rec map[string]interface{}, key string) bool {
	v, ok := rec[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// parseCodeSuffix converts the numeric suffix from a NAMES.DBF code string
// (e.g. "CC001" → 1, "PY02" → 2) to an integer.
func parseCodeSuffix(suffix string) int {
	num := 0
	for _, ch := range suffix {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		}
	}
	return num
}

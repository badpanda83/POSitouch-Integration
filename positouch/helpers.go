// Package positouch provides typed readers for POSitouch DBF data files.
// helpers.go contains shared field-extraction utilities used across all readers.
package positouch

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
func parseCodeSuffix(suffix string) int {
	num := 0
	for _, ch := range suffix {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		}
	}
	return num
}

package positouch

// helpers.go — shared field-extraction helpers used by all positouch readers.

// stringField extracts a string value from a DBF record map.
// Returns an empty string if the key is missing or the value is not a string.
func stringField(r map[string]interface{}, key string) string {
	v, ok := r[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// intField extracts an integer value from a DBF record map.
// DBF numeric fields with no decimals are stored as int by the reader.
// Returns 0 if the key is missing, nil, or not an int.
func intField(r map[string]interface{}, key string) int {
	v, ok := r[key]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}
	return 0
}

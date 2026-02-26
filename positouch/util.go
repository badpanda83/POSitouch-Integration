package positouch

import (
	"os"
	"path/filepath"
	"strings"
)

// findDBF performs a case-insensitive search for filename inside dir.
// Returns the full path if found, or empty string if not found.
func findDBF(dir, filename string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	upper := strings.ToUpper(filename)
	for _, e := range entries {
		if strings.ToUpper(e.Name()) == upper {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// toFloat64 converts a DBF numeric value to float64, returning 0 on failure.
func toFloat64(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// toString converts a DBF character value to string, returning "" on failure.
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

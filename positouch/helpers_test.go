package positouch

import (
	"testing"
)

func TestParseCodeSuffix_ValidNumeric(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"001", 1},
		{"02", 2},
		{"10", 10},
		{"100", 100},
		{"0", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := parseCodeSuffix(c.input)
		if got != c.want {
			t.Errorf("parseCodeSuffix(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestParseCodeSuffix_NonNumeric(t *testing.T) {
	// Non-numeric suffixes should return 0 and log a warning (no panic).
	got := parseCodeSuffix("0A3")
	if got != 0 {
		t.Errorf("parseCodeSuffix(%q) = %d, want 0", "0A3", got)
	}
}

func TestFloatField_Present(t *testing.T) {
	rec := map[string]interface{}{"VAL": float64(42)}
	if got := floatField(rec, "VAL"); got != 42 {
		t.Errorf("floatField = %v, want 42", got)
	}
}

func TestFloatField_Absent(t *testing.T) {
	rec := map[string]interface{}{}
	if got := floatField(rec, "MISSING"); got != 0 {
		t.Errorf("floatField absent = %v, want 0", got)
	}
}

func TestFloatField_WrongType(t *testing.T) {
	rec := map[string]interface{}{"VAL": "not-a-float"}
	if got := floatField(rec, "VAL"); got != 0 {
		t.Errorf("floatField wrong type = %v, want 0", got)
	}
}

func TestStringField_Present(t *testing.T) {
	rec := map[string]interface{}{"NAME": "Alice"}
	if got := stringField(rec, "NAME"); got != "Alice" {
		t.Errorf("stringField = %q, want %q", got, "Alice")
	}
}

func TestStringField_Absent(t *testing.T) {
	rec := map[string]interface{}{}
	if got := stringField(rec, "MISSING"); got != "" {
		t.Errorf("stringField absent = %q, want empty", got)
	}
}

func TestStringField_WrongType(t *testing.T) {
	rec := map[string]interface{}{"NAME": 99}
	if got := stringField(rec, "NAME"); got != "" {
		t.Errorf("stringField wrong type = %q, want empty", got)
	}
}

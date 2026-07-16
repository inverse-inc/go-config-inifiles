package inifiles

import (
	"reflect"
	"testing"
)

// translated from config-inifiles/t/11copy-ties-with-array-vals.t
func TestCopyTiesWithArrayVals(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "array.ini"))
	if err != nil {
		t.Fatalf("NewFromFile array.ini: %v", err)
	}

	got, ok := cfg.Vals("Sect", "Par")
	if !ok {
		t.Fatalf("Vals(Sect,Par) not found")
	}
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Vals(Sect,Par) = %v, want %v", got, want)
	}

	// Vals returns a fresh slice copy; mutating it must not affect the config.
	copySlice := append([]string(nil), got...)
	copySlice[1] = "A"

	got2, ok := cfg.Vals("Sect", "Par")
	if !ok {
		t.Fatalf("Vals(Sect,Par) after mutation not found")
	}
	if !reflect.DeepEqual(got2, want) {
		t.Errorf("Vals(Sect,Par) after mutating returned slice = %v, want %v (unaffected)", got2, want)
	}
}

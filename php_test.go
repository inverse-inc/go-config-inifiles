package inifiles

import (
	"reflect"
	"testing"
)

// translated from config-inifiles/t/36php-compat.t
func TestPHPCompat(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "php-compat.ini"), WithPHPCompat(true))
	if err != nil {
		t.Fatalf("NewFromFile php-compat.ini: %v", err)
	}

	// test 1: double-quoted string with escapes collapses to str"ing.
	if v, ok := cfg.Val("group1", "val1"); !ok || v != `str"ing` {
		t.Errorf(`Val(group1,val1) = %q ok=%v, want "str\"ing"`, v, ok)
	}

	// test 2: single-quoted string with doubled quote collapses to string.
	if v, ok := cfg.Val("group1", "val2"); !ok || v != "string" {
		t.Errorf(`Val(group1,val2) = %q ok=%v, want "string"`, v, ok)
	}

	// test 3: php array key (val1[]) accumulates multiple values.
	got, ok := cfg.Vals("group2", "val1")
	if !ok {
		t.Fatalf("Vals(group2,val1) not found")
	}
	want := []string{"1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Vals(group2,val1) = %v, want %v", got, want)
	}
}

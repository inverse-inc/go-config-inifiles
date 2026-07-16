package inifiles

import "testing"

// translated from config-inifiles/t/13val-in-list-context.t
//
// A missing parameter queried in list context (Go: Vals) must return a
// false/empty result, NOT a one-element list. Mirrors Perl `my @v = val(...)`
// being false in boolean context when the param is absent.
func TestValInListContext_MissingIsFalse(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "array.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	vs, ok := cfg.Vals("Sect", "NotExist")
	if ok {
		t.Errorf("Vals(Sect,NotExist) ok=true, want false for a missing parameter")
	}
	if vs != nil {
		t.Errorf("Vals(Sect,NotExist) = %v, want nil (empty list, not a one-element list)", vs)
	}

	// Sanity: an existing multi-value param in the same file returns its
	// elements (ensures the file actually parsed).
	got, ok := cfg.Vals("Sect", "Par")
	if !ok || len(got) != 3 || got[0] != "1" || got[1] != "2" || got[2] != "3" {
		t.Fatalf(`Vals(Sect,Par) = %v ok=%v, want [1 2 3]`, got, ok)
	}
}

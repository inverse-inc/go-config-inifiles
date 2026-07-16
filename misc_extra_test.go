package inifiles

import (
	"testing"
)

// SetParameterEOT is a no-op for empty inputs or a missing parameter.
func TestSetParameterEOT_NoopCases(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	for _, args := range []struct{ s, p, e string }{
		{"", "p", "EOT"},
		{"S", "", "EOT"},
		{"S", "p", ""},
	} {
		if err := cfg.SetParameterEOT(args.s, args.p, args.e); err != nil {
			t.Errorf("SetParameterEOT(%q,%q,%q) = %v, want nil", args.s, args.p, args.e, err)
		}
	}
	// Missing parameter: no-op, no EOT recorded.
	if err := cfg.SetParameterEOT("S", "ghost", "EOT"); err != nil {
		t.Errorf("SetParameterEOT(S,ghost,EOT) = %v, want nil", err)
	}
	if got := cfg.GetParameterEOT("S", "ghost"); got != "" {
		t.Errorf("GetParameterEOT(S,ghost) = %q, want empty", got)
	}
}

// SetParameterEOT sets an explicit marker that GetParameterEOT returns.
func TestSetParameterEOT_RoundTrip(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if err := cfg.SetParameterEOT("S", "p", "MARK"); err != nil {
		t.Fatalf("SetParameterEOT: %v", err)
	}
	if got := cfg.GetParameterEOT("S", "p"); got != "MARK" {
		t.Errorf("GetParameterEOT = %q, want MARK", got)
	}
}

// GetParameterEOT returns "EOT" for a multi-valued param with no explicit EOT
// (the isMulti default branch), and "" for a scalar with no EOT.
func TestGetParameterEOT_Defaults(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "multi", "a", "b") // multi-valued, hasEOT auto-set to true via NewVal
	if got := cfg.GetParameterEOT("S", "multi"); got != "EOT" {
		t.Errorf("GetParameterEOT(multi) = %q, want EOT", got)
	}
	// Scalar with no EOT -> "".
	cfg.NewVal("S", "scalar", "x")
	if got := cfg.GetParameterEOT("S", "scalar"); got != "" {
		t.Errorf("GetParameterEOT(scalar) = %q, want empty", got)
	}
	// Missing param/section -> "".
	if got := cfg.GetParameterEOT("NoSect", "p"); got != "" {
		t.Errorf("GetParameterEOT(NoSect,p) = %q, want empty", got)
	}
}

// DeleteParameterEOT clears the explicit marker; a multi-valued param reverts
// to "EOT" on output. Missing param is a no-op; the param must exist.
func TestDeleteParameterEOT(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	cfg.SetParameterEOT("S", "p", "MARK")
	if err := cfg.DeleteParameterEOT("S", "p"); err != nil {
		t.Fatalf("DeleteParameterEOT: %v", err)
	}
	if got := cfg.GetParameterEOT("S", "p"); got != "" {
		t.Errorf("GetParameterEOT after delete = %q, want empty", got)
	}
	// Missing param: no-op, no panic.
	if err := cfg.DeleteParameterEOT("S", "ghost"); err != nil {
		t.Errorf("DeleteParameterEOT(S,ghost) = %v, want nil", err)
	}
	if err := cfg.DeleteParameterEOT("NoSect", "p"); err != nil {
		t.Errorf("DeleteParameterEOT(NoSect,p) = %v, want nil", err)
	}
}

// value helpers direct coverage.
func TestValueHelpers(t *testing.T) {
	// nil-lines value: scalar "", slice nil, isNil true.
	var v value
	if !v.isNil() {
		t.Errorf("value{} isNil = false, want true")
	}
	if got := v.scalar(); got != "" {
		t.Errorf("value{} scalar = %q, want empty", got)
	}
	if got := v.slice(); got != nil {
		t.Errorf("value{} slice = %v, want nil", got)
	}
	// single
	sv := singleValue("x")
	if sv.isNil() {
		t.Errorf("singleValue isNil = true, want false")
	}
	if got := sv.scalar(); got != "x" {
		t.Errorf("singleValue scalar = %q, want x", got)
	}
	if got := sv.slice(); !sliceEq(got, []string{"x"}) {
		t.Errorf("singleValue slice = %v, want [x]", got)
	}
	// multi join with newline
	mv := multiValue([]string{"a", "b", "c"})
	if got := mv.scalar(); got != "a\nb\nc" {
		t.Errorf("multiValue scalar = %q, want a\\nb\\nc", got)
	}
	if got := mv.slice(); !sliceEq(got, []string{"a", "b", "c"}) {
		t.Errorf("multiValue slice = %v, want [a b c]", got)
	}
	// slice returns a fresh copy.
	got := mv.slice()
	got[0] = "X"
	again := mv.slice()
	if again[0] != "a" {
		t.Errorf("multiValue.slice returned non-copy; second call[0] = %q, want a", again[0])
	}
}

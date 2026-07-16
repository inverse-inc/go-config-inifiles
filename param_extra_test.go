package inifiles

import (
	"errors"
	"testing"
)

// SetVal returns ErrNotFound when the section is missing.
func TestSetVal_MissingSection(t *testing.T) {
	cfg := New()
	if err := cfg.SetVal("NoSect", "p", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetVal(NoSect,p,v) = %v, want ErrNotFound", err)
	}
}

// SetVal returns ErrNotFound when the parameter is missing.
func TestSetVal_MissingParam(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if err := cfg.SetVal("S", "ghost", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetVal(S,ghost,v) = %v, want ErrNotFound", err)
	}
}

// SetVal with zero values sets an empty scalar (covers the len(vals)==0 branch).
func TestSetVal_ZeroValues(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "old")
	if err := cfg.SetVal("S", "p"); err != nil {
		t.Fatalf("SetVal(S,p): %v", err)
	}
	v, ok := cfg.Val("S", "p")
	if !ok || v != "" {
		t.Errorf("Val(S,p) = (%q,%v), want (\"\",true)", v, ok)
	}
}

// SetVal with multiple values produces a multi-valued param with an EOT marker.
func TestSetVal_MultipleValues(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "old")
	if err := cfg.SetVal("S", "p", "a", "b"); err != nil {
		t.Fatalf("SetVal(S,p,a,b): %v", err)
	}
	vals, ok := cfg.Vals("S", "p")
	if !ok || len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Fatalf("Vals(S,p) = %v ok=%v, want [a b]", vals, ok)
	}
	if got := cfg.GetParameterEOT("S", "p"); got != "EOT" {
		t.Errorf("GetParameterEOT(S,p) = %q, want EOT", got)
	}
}

// SetVal under PHPCompat resolves a "[]"-suffixed name to the stripped param
// (covers the stripPHPTailArray branch in SetVal and findParam).
func TestSetVal_PHPCompatStripTail(t *testing.T) {
	cfg := New(WithPHPCompat(true))
	cfg.NewVal("S", "arr", "v")
	if err := cfg.SetVal("S", "arr[]", "new"); err != nil {
		t.Fatalf("SetVal(S,arr[],new): %v", err)
	}
	v, ok := cfg.Val("S", "arr[]")
	if !ok || v != "new" {
		t.Errorf("Val(S,arr[]) = (%q,%v), want (new,true)", v, ok)
	}
}

// DelVal is a no-op for empty names (covers the early-return branch).
func TestDelVal_EmptyInputs(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if err := cfg.DelVal("", "p"); err != nil {
		t.Errorf("DelVal(\"\",p) = %v, want nil", err)
	}
	if err := cfg.DelVal("S", ""); err != nil {
		t.Errorf("DelVal(S,\"\") = %v, want nil", err)
	}
	if _, ok := cfg.Val("S", "p"); !ok {
		t.Errorf("Val(S,p) gone after empty DelVal")
	}
}

// DelVal on a missing section or parameter is a no-op (covers both no-op paths).
func TestDelVal_Missing(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if err := cfg.DelVal("NoSect", "p"); err != nil {
		t.Errorf("DelVal(NoSect,p) = %v, want nil", err)
	}
	if err := cfg.DelVal("S", "ghost"); err != nil {
		t.Errorf("DelVal(S,ghost) = %v, want nil", err)
	}
	if _, ok := cfg.Val("S", "p"); !ok {
		t.Errorf("Val(S,p) gone after no-op DelVal")
	}
}

// DelVal under PHPCompat strips a "[]"-suffixed name before deleting.
func TestDelVal_PHPCompatStripTail(t *testing.T) {
	cfg := New(WithPHPCompat(true))
	cfg.NewVal("S", "arr", "v")
	if err := cfg.DelVal("S", "arr[]"); err != nil {
		t.Fatalf("DelVal(S,arr[]): %v", err)
	}
	if cfg.Exists("S", "arr") {
		t.Errorf("Exists(S,arr) = true after DelVal(S,arr[])")
	}
}

// DelVal records a tombstone (covered via delta output already); verify it drops
// the live param and the param is gone from normal output.
func TestDelVal_RemovesFromParamsAndNames(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "a", "1")
	cfg.NewVal("S", "b", "2")
	if err := cfg.DelVal("S", "a"); err != nil {
		t.Fatalf("DelVal(S,a): %v", err)
	}
	names := cfg.Parameters("S")
	if len(names) != 1 || names[0] != "b" {
		t.Errorf("Parameters(S) after DelVal = %v, want [b]", names)
	}
}

// PushVal returns ErrNotFound for a missing section and missing parameter.
func TestPushVal_Missing(t *testing.T) {
	cfg := New()
	if err := cfg.PushVal("NoSect", "p", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("PushVal(NoSect,p,v) = %v, want ErrNotFound", err)
	}
	cfg.NewVal("S", "p", "v")
	if err := cfg.PushVal("S", "ghost", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("PushVal(S,ghost,v) = %v, want ErrNotFound", err)
	}
}

// PushVal with no values is a no-op (covers the len(vals)==0 branch).
func TestPushVal_NoValues(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if err := cfg.PushVal("S", "p"); err != nil {
		t.Errorf("PushVal(S,p) no vals = %v, want nil", err)
	}
	v, ok := cfg.Val("S", "p")
	if !ok || v != "v" {
		t.Errorf("Val(S,p) after no-op PushVal = (%q,%v), want (v,true)", v, ok)
	}
}

// PushVal onto a scalar accumulates into a multi-valued param, exercising the
// empty-lines fallback when the initial scalar had an empty value.
func TestPushVal_AccumulateFromEmptyScalar(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "")
	if err := cfg.PushVal("S", "p", "x", "y"); err != nil {
		t.Fatalf("PushVal(S,p,x,y): %v", err)
	}
	vals, ok := cfg.Vals("S", "p")
	if !ok {
		t.Fatalf("Vals(S,p) not found")
	}
	if len(vals) < 2 || vals[len(vals)-1] != "y" {
		t.Errorf("Vals(S,p) = %v, want last element y", vals)
	}
}

// findParam returns nil for empty section and a missing section.
func TestFindParam_Missing(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if p := cfg.findParam("", "p"); p != nil {
		t.Errorf("findParam(\"\",p) = %v, want nil", p)
	}
	if p := cfg.findParam("NoSect", "p"); p != nil {
		t.Errorf("findParam(NoSect,p) = %v, want nil", p)
	}
	if p := cfg.findParam("S", "ghost"); p != nil {
		t.Errorf("findParam(S,ghost) = %v, want nil", p)
	}
}

// findParam under PHPCompat strips "[]" from the lookup name.
func TestFindParam_PHPCompatStripTail(t *testing.T) {
	cfg := New(WithPHPCompat(true))
	cfg.NewVal("S", "arr", "v")
	if p := cfg.findParam("S", "arr[]"); p == nil {
		t.Errorf("findParam(S,arr[]) = nil, want found (stripped)")
	}
}

// Val/ValWithDefault/Vals with empty inputs return zero/nil,false.
func TestVal_EmptyInputs(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if _, ok := cfg.Val("", "p"); ok {
		t.Errorf("Val(\"\",p) ok=true, want false")
	}
	if _, ok := cfg.Val("S", ""); ok {
		t.Errorf("Val(S,\"\") ok=true, want false")
	}
	if v := cfg.ValWithDefault("NoSect", "ghost", "def"); v != "def" {
		t.Errorf("ValWithDefault(NoSect,ghost,def) = %q, want def", v)
	}
	if _, ok := cfg.Vals("", "p"); ok {
		t.Errorf("Vals(\"\",p) ok=true, want false")
	}
}

// Exists returns false for empty inputs.
func TestExists_Empty(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if cfg.Exists("", "p") {
		t.Errorf("Exists(\"\",p) = true, want false")
	}
	if cfg.Exists("S", "") {
		t.Errorf("Exists(S,\"\") = true, want false")
	}
	if cfg.Exists("S", "ghost") {
		t.Errorf("Exists(S,ghost) = true, want false")
	}
	if cfg.Exists("NoSect", "p") {
		t.Errorf("Exists(NoSect,p) = true, want false")
	}
	if !cfg.Exists("S", "p") {
		t.Errorf("Exists(S,p) = false, want true")
	}
}

// NewVal with empty inputs is a no-op, and a single value + multi value paths.
func TestNewVal_EdgeBranches(t *testing.T) {
	cfg := New()
	if err := cfg.NewVal("", "p", "v"); err != nil {
		t.Errorf("NewVal(\"\",p,v) = %v, want nil", err)
	}
	if err := cfg.NewVal("S", "", "v"); err != nil {
		t.Errorf("NewVal(S,\"\",v) = %v, want nil", err)
	}
	// Zero values -> single empty value.
	if err := cfg.NewVal("S", "z"); err != nil {
		t.Errorf("NewVal(S,z): %v", err)
	}
	if v, ok := cfg.Val("S", "z"); !ok || v != "" {
		t.Errorf("Val(S,z) = (%q,%v), want (\"\",true)", v, ok)
	}
}

// NewVal overwrites an existing in-section param (covers the replace branch).
func TestNewVal_ReplaceExisting(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "old")
	if err := cfg.NewVal("S", "p", "new"); err != nil {
		t.Fatalf("NewVal(S,p,new): %v", err)
	}
	v, ok := cfg.Val("S", "p")
	if !ok || v != "new" {
		t.Errorf("Val(S,p) after overwrite = (%q,%v), want (new,true)", v, ok)
	}
	// Only one p entry remains.
	names := cfg.Parameters("S")
	n := 0
	for _, nm := range names {
		if nm == "p" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("Parameters(S) has %d p entries, want 1", n)
	}
}

// NewVal under PHPCompat strips "[]" from the param name on creation.
func TestNewVal_PHPCompatStripTail(t *testing.T) {
	cfg := New(WithPHPCompat(true))
	cfg.NewVal("S", "arr[]", "v")
	if _, ok := cfg.Val("S", "arr"); !ok {
		t.Errorf("Val(S,arr) not found after NewVal(S,arr[])")
	}
}

// Val with the default-section fallback returns the default's value (covers
// the default-section branch of lookupParamWithFallback and defaultSect != "").
func TestVal_DefaultSectionFallback(t *testing.T) {
	cfg := New(WithDefault("defaults"))
	cfg.NewVal("defaults", "shared", "fromdefault")
	cfg.NewVal("S", "local", "local")

	v, ok := cfg.Val("S", "shared") // missing in S, falls back to defaults
	if !ok || v != "fromdefault" {
		t.Errorf("Val(S,shared) = (%q,%v), want (fromdefault,true)", v, ok)
	}
	// Asking the default section directly should still work and not recurse.
	v, ok = cfg.Val("defaults", "shared")
	if !ok || v != "fromdefault" {
		t.Errorf("Val(defaults,shared) = (%q,%v), want (fromdefault,true)", v, ok)
	}
	// A key missing from both returns false.
	if _, ok := cfg.Val("S", "missing"); ok {
		t.Errorf("Val(S,missing) ok=true, want false")
	}
}

// Default-section fallback is case-insensitive against the requested section
// (the guard compares c.caseify(sect) != c.defaultSect).
func TestVal_DefaultFallbackNotTriggeredForDefaultSectItself(t *testing.T) {
	cfg := New(WithDefault("defaults"))
	cfg.NewVal("defaults", "shared", "v")
	// Requesting "defaults" directly should find it without fallback.
	v, ok := cfg.Val("defaults", "shared")
	if !ok || v != "v" {
		t.Errorf("Val(defaults,shared) = (%q,%v), want (v,true)", v, ok)
	}
}

package inifiles

import (
	"testing"
)

// SetParameterComment on a non-existent parameter auto-creates a placeholder
// param (and the section if needed) holding the comment (covers the create-param
// branch in comments.go).
func TestSetParameterComment_OnNonExistentParam(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "existing", "v")

	// Comment for a param that doesn't exist yet: creates the placeholder.
	n, err := cfg.SetParameterComment("S", "ghost", "a comment")
	if err != nil {
		t.Fatalf("SetParameterComment: %v", err)
	}
	if n != 1 {
		t.Fatalf("returned %d, want 1", n)
	}
	got, ok := cfg.GetParameterComment("S", "ghost")
	if !ok {
		t.Fatalf("GetParameterComment(S,ghost) not found after set")
	}
	if !sliceEq(got, []string{"# a comment"}) {
		t.Errorf("GetParameterComment(S,ghost) = %v, want [# a comment]", got)
	}
	// The placeholder param exists with an empty value (the placeholder marker).
	v, ok := cfg.Val("S", "ghost")
	if !ok {
		t.Fatalf("Val(S,ghost) not found, want found")
	}
	if v != "" {
		t.Errorf("Val(S,ghost) = %q, want empty (placeholder)", v)
	}
}

// SetParameterComment for a param in a section that doesn't exist: section is
// created by AddSection, then the placeholder param is created.
func TestSetParameterComment_OnNonExistentSection(t *testing.T) {
	cfg := New()
	if _, err := cfg.SetParameterComment("NewSect", "ghost", "a comment"); err != nil {
		t.Fatalf("SetParameterComment: %v", err)
	}
	if !cfg.SectionExists("NewSect") {
		t.Errorf("section NewSect should have been created")
	}
	got, ok := cfg.GetParameterComment("NewSect", "ghost")
	if !ok || !sliceEq(got, []string{"# a comment"}) {
		t.Errorf("GetParameterComment = %v ok=%v", got, ok)
	}
}

// empty inputs to SetParameterComment are no-ops (early return branch).
func TestSetParameterComment_EmptyInputs(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")
	if n, err := cfg.SetParameterComment("S", "p"); err != nil || n != 0 {
		t.Errorf("SetParameterComment(S,p) no comments = (n=%d err=%v), want (0,nil)", n, err)
	}
	if n, err := cfg.SetParameterComment("", "p", "c"); err != nil || n != 0 {
		t.Errorf("SetParameterComment(\"\",p,c) = (n=%d err=%v), want (0,nil)", n, err)
	}
	if n, err := cfg.SetParameterComment("S", "", "c"); err != nil || n != 0 {
		t.Errorf("SetParameterComment(S,\"\",c) = (n=%d err=%v), want (0,nil)", n, err)
	}
}

// Under PHPCompat, SetParameterComment strips a trailing "[]" from the param
// name when creating the placeholder (covers the PHPCompat branch).
func TestSetParameterComment_PHPCompatStripTail(t *testing.T) {
	cfg := New(WithPHPCompat(true))
	if _, err := cfg.SetParameterComment("S", "arr[]", "c"); err != nil {
		t.Fatalf("SetParameterComment: %v", err)
	}
	if _, ok := cfg.GetParameterComment("S", "arr"); !ok {
		t.Errorf("GetParameterComment(S,arr) not found, want found (stripped)")
	}
	// "arr[]" also resolves to the stripped name under PHPCompat.
	if _, ok := cfg.GetParameterComment("S", "arr[]"); !ok {
		t.Errorf("GetParameterComment(S,arr[]) not found, want found (lookup also strips)")
	}
	// The actual stored param name is the stripped form.
	if p := cfg.findParam("S", "arr"); p == nil {
		t.Errorf("findParam(S,arr) = nil, want placeholder param")
	}
}

// SetParameterTrailingComment errors when the parameter doesn't exist, and is
// a no-op when sect, parm, or comment is empty.
func TestSetParameterTrailingComment_ErrorAndNoop(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")

	if err := cfg.SetParameterTrailingComment("S", "ghost", "c"); err == nil {
		t.Errorf("SetParameterTrailingComment on missing param: want error, got nil")
	}
	for _, args := range []struct{ s, p, c string }{
		{"", "p", "c"},
		{"S", "", "c"},
		{"S", "p", ""},
	} {
		if err := cfg.SetParameterTrailingComment(args.s, args.p, args.c); err != nil {
			t.Errorf("SetParameterTrailingComment(%q,%q,%q) = err %v, want nil", args.s, args.p, args.c, err)
		}
	}
}

// GetParameterTrailingComment reports absence via the boolean and returns the
// stored trailing for an existing param.
func TestGetParameterTrailingComment_AbsenceAndRoundTrip(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "p", "v")

	if _, ok := cfg.GetParameterTrailingComment("S", "ghost"); ok {
		t.Errorf("GetParameterTrailingComment(S,ghost) ok=true, want false")
	}
	if _, ok := cfg.GetParameterTrailingComment("NoSect", "p"); ok {
		t.Errorf("GetParameterTrailingComment(NoSect,p) ok=true, want false")
	}
	if err := cfg.SetParameterTrailingComment("S", "p", "hi"); err != nil {
		t.Fatalf("SetParameterTrailingComment: %v", err)
	}
	if got, ok := cfg.GetParameterTrailingComment("S", "p"); !ok || got != "hi" {
		t.Errorf("GetParameterTrailingComment(S,p) = (%q,%v), want (hi,true)", got, ok)
	}
}

// DeleteParameterComment on a missing param/section is a no-op (no panic).
func TestDeleteParameterComment_Noop(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteParameterComment("NoSect", "p"); err != nil {
		t.Errorf("DeleteParameterComment(NoSect,p) = %v, want nil", err)
	}
	cfg.NewVal("S", "p", "v")
	if err := cfg.DeleteParameterComment("S", "ghost"); err != nil {
		t.Errorf("DeleteParameterComment(S,ghost) = %v, want nil", err)
	}
}

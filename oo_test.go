package inifiles

import (
	"os"
	"path/filepath"
	"testing"
)

// translated from config-inifiles/t/06oo.t
func TestOO(t *testing.T) {
	// test 1: en has no groups.
	en, err := NewFromFile(testdata(t, "en.ini"))
	if err != nil {
		t.Fatalf("NewFromFile en.ini: %v", err)
	}
	if g := en.Groups(); len(g) != 0 {
		t.Fatalf("en.Groups() = %v, want empty", g)
	}

	// test 2: new object is non-nil.
	ini := New()
	if ini == nil {
		t.Fatalf("New() returned nil")
	}

	// test 3: set file name.
	tmp := filepath.Join(t.TempDir(), "test06.ini")
	if err := ini.SetFileName(tmp); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}

	// test 4: rewrite creates the file.
	if err := ini.Rewrite(); err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if info, err := os.Stat(tmp); err != nil || info.IsDir() {
		t.Fatalf("Rewrite should create %s", tmp)
	}

	// test 5: SetSectionComment succeeds.
	if err := ini.NewVal("Section1", "Parameter1", "Value1"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	n, err := ini.SetSectionComment("Section1",
		"Line 1 of section comment.",
		"Line 2 of section comment",
		"Line 3 of section comment",
	)
	if err != nil || n == 0 {
		t.Fatalf("SetSectionComment: n=%d err=%v", n, err)
	}

	// test 6: GetSectionComment returns prefixed lines.
	cmt, ok := ini.GetSectionComment("Section1")
	if !ok {
		t.Fatalf("GetSectionComment: not found")
	}
	wantCmt := []string{
		"# Line 1 of section comment.",
		"# Line 2 of section comment",
		"# Line 3 of section comment",
	}
	if len(cmt) != len(wantCmt) {
		t.Fatalf("GetSectionComment = %v, want %v", cmt, wantCmt)
	}
	for i := range wantCmt {
		if cmt[i] != wantCmt[i] {
			t.Errorf("GetSectionComment[%d] = %q, want %q", i, cmt[i], wantCmt[i])
		}
	}

	// test 7: DeleteSectionComment.
	if err := ini.DeleteSectionComment("Section1"); err != nil {
		t.Fatalf("DeleteSectionComment: %v", err)
	}
	if _, ok := ini.GetSectionComment("Section1"); ok {
		t.Fatalf("GetSectionComment after delete should be false")
	}

	// test 8: CopySection.
	if err := ini.CopySection("Section1", "Section2", false); err != nil {
		t.Fatalf("CopySection: %v", err)
	}
	if p := ini.Parameters("Section2"); len(p) == 0 {
		t.Fatalf("Parameters(Section2) empty, want non-empty after CopySection")
	}

	// test 9: DeleteSection.
	if err := ini.DeleteSection("Section1"); err != nil {
		t.Fatalf("DeleteSection: %v", err)
	}
	if p := ini.Parameters("Section1"); len(p) != 0 {
		t.Fatalf("Parameters(Section1) = %v, want empty after delete", p)
	}

	// test 10: RenameSection.
	if err := ini.RenameSection("Section2", "Section1", false); err != nil {
		t.Fatalf("RenameSection: %v", err)
	}
	if p := ini.Parameters("Section2"); len(p) != 0 {
		t.Errorf("Parameters(Section2) = %v, want empty after rename", p)
	}
	if p := ini.Parameters("Section1"); len(p) == 0 {
		t.Errorf("Parameters(Section1) empty, want non-empty after rename")
	}
	if v, ok := ini.Val("Section1", "Parameter1"); !ok || v != "Value1" {
		t.Errorf(`Val(Section1,Parameter1) = %q ok=%v, want "Value1"`, v, ok)
	}

	// test 11: Delete entire config.
	ini.Delete()
	if s := ini.Sections(); len(s) != 0 {
		t.Fatalf("Sections() = %v, want empty after Delete()", s)
	}
}

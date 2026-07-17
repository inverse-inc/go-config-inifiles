package inifiles

import (
	"path/filepath"
	"strings"
	"testing"
)

// Translated from config-inifiles/t/03comments.t.
func TestComments_PreservationAndAccess(t *testing.T) {
	// Load ini file and write as new file, then re-read (tests 1-3).
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile test.ini: %v", err)
	}

	tmpPath := filepath.Join(t.TempDir(), "test03.ini")
	if err := cfg.SetFileName(tmpPath); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	// Perl test uses SetWriteMode("0666"); the Go port accepts an os.FileMode.
	cfg.SetWriteMode(0666)
	if err := cfg.WriteFile(tmpPath); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := NewFromFile(tmpPath); err != nil {
		t.Fatalf("re-read test03.ini: %v", err)
	}

	// Inspect the written contents directly.
	var buf strings.Builder
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	contents := buf.String()

	// TEST 2: section comment preserved.
	if !strings.Contains(contents, "# This is a section comment\n[test1]") {
		t.Errorf("section comment not preserved; contents=\n%s", contents)
	}

	// TEST 3: parameter comment preserved.
	if !strings.Contains(contents, "# This is a parm comment\nfive=value5") {
		t.Errorf("parm comment not preserved; contents=\n%s", contents)
	}

	// Setting Section Comment (test 4): setval('foo','bar','qux') then comment.
	if err := cfg.NewVal("foo", "bar", "qux"); err != nil {
		t.Fatalf("NewVal foo/bar: %v", err)
	}
	n, err := cfg.SetSectionComment("foo",
		"This is a section comment",
		"This comment takes two lines!")
	if err != nil {
		t.Fatalf("SetSectionComment: %v", err)
	}
	if n != 2 {
		t.Errorf("SetSectionComment returned %d, want 2", n)
	}

	// TEST 5: Getting Section Comment (list).
	got, ok := cfg.GetSectionComment("foo")
	if !ok {
		t.Fatalf("GetSectionComment(foo) not found")
	}
	want := []string{"# This is a section comment", "# This comment takes two lines!"}
	if !sliceEq(got, want) {
		t.Errorf("GetSectionComment(foo) = %v, want %v", got, want)
	}

	// TEST 6: scalar context join (joined by input record separator "\n").
	if joined := strings.Join(got, "\n"); joined != "# This is a section comment\n# This comment takes two lines!" {
		t.Errorf("joined section comment = %q", joined)
	}

	// TEST 7: Deleting Section Comment.
	if err := cfg.DeleteSectionComment("foo"); err != nil {
		t.Fatalf("DeleteSectionComment: %v", err)
	}
	if _, ok := cfg.GetSectionComment("foo"); ok {
		t.Errorf("foo section comment should not exist after delete")
	}

	// TEST 8: Setting Parameter Comment.
	if _, err := cfg.SetParameterComment("foo", "bar",
		"This is a parameter comment",
		"This comment takes two lines!"); err != nil {
		t.Fatalf("SetParameterComment: %v", err)
	}

	// TEST 9: Getting Parameter Comment (list).
	got, ok = cfg.GetParameterComment("foo", "bar")
	if !ok {
		t.Fatalf("GetParameterComment(foo,bar) not found")
	}
	want = []string{"# This is a parameter comment", "# This comment takes two lines!"}
	if !sliceEq(got, want) {
		t.Errorf("GetParameterComment(foo,bar) = %v, want %v", got, want)
	}

	// TEST 10: scalar context join.
	if joined := strings.Join(got, "\n"); joined != "# This is a parameter comment\n# This comment takes two lines!" {
		t.Errorf("joined parm comment = %q", joined)
	}

	// TEST 11: Deleting Parameter Comment.
	if err := cfg.DeleteParameterComment("foo", "bar"); err != nil {
		t.Fatalf("DeleteParameterComment: %v", err)
	}
	if _, ok := cfg.GetParameterComment("foo", "bar"); ok {
		t.Errorf("GetParameterComment(foo,bar) should not exist after delete")
	}

	// TEST 12: Reading a section comment from the file.
	got, ok = cfg.GetSectionComment("test1")
	if !ok {
		t.Fatalf("GetSectionComment(test1) not found")
	}
	if !sliceEq(got, []string{"# This is a section comment"}) {
		t.Errorf("GetSectionComment(test1) = %v, want [# This is a section comment]", got)
	}

	// TEST 13: Reading a parameter comment from the file.
	got, ok = cfg.GetParameterComment("test2", "five")
	if !ok {
		t.Fatalf("GetParameterComment(test2,five) not found")
	}
	if !sliceEq(got, []string{"# This is a parm comment"}) {
		t.Errorf("GetParameterComment(test2,five) = %v, want [# This is a parm comment]", got)
	}

	// TEST 14: Reading a comment that starts with ';'.
	got, ok = cfg.GetSectionComment("MixedCaseSect")
	if !ok {
		t.Fatalf("GetSectionComment(MixedCaseSect) not found")
	}
	if !sliceEq(got, []string{"; This is a semi-colon comment"}) {
		t.Errorf("GetSectionComment(MixedCaseSect) = %v, want [; This is a semi-colon comment]", got)
	}
}

// Translated from config-inifiles/t/03comments.t tests 15-17 (cmt.ini with
// alternate comment char '@' and line continuation).
func TestComments_AlternateCharAndContinuation(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "cmt.ini"),
		WithCommentChar('@'),
		WithAllowContinue(true))
	if err != nil {
		t.Fatalf("NewFromFile cmt.ini: %v", err)
	}

	// TEST 16: parameter comment starting with '@' at-sign.
	got, ok := cfg.GetParameterComment("Library", "addmultf_lib")
	if !ok {
		t.Fatalf("GetParameterComment(Library,addmultf_lib) not found")
	}
	needle := "@#@CF Automatically created by 'config_project' at Thu Mar 21 08:46:54 2002"
	found := false
	for _, line := range got {
		if strings.Contains(line, needle) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("parameter comment missing %q; got %v", needle, got)
	}

	// TEST 17: continued value joins two lines via backslash.
	v, ok := cfg.Val("turbo_library", "TurboLibPaths")
	if !ok {
		t.Fatalf("Val(turbo_library,TurboLibPaths) not found")
	}
	if !strings.Contains(v, "$WORKAREA/resources/c11_test_flow/vhdl_rtl") ||
		!strings.Contains(v, "$WORKAREA/resources/cstarlib_reg_1v5/vhdl_rtl") {
		t.Errorf("continued value = %q, want both paths separated by whitespace", v)
	}
	joined := strings.Join(
		strings.Fields(v), " ")
	if !strings.Contains(joined, "$WORKAREA/resources/c11_test_flow/vhdl_rtl $WORKAREA/resources/cstarlib_reg_1v5/vhdl_rtl") {
		t.Errorf("continued value fields = %q", joined)
	}
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

package inifiles

import (
	"bytes"
	"strings"
	"testing"
)

// Exercises PushVal: scalar -> array append.
func TestPushVal(t *testing.T) {
	cfg := New()
	if err := cfg.NewVal("s", "k", "a"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.PushVal("s", "k", "b", "c"); err != nil {
		t.Fatal(err)
	}
	vs, _ := cfg.Vals("s", "k")
	want := []string{"a", "b", "c"}
	if len(vs) != 3 || vs[0] != want[0] || vs[1] != want[1] || vs[2] != want[2] {
		t.Fatalf("Vals=%v want %v", vs, want)
	}
	if v, _ := cfg.Val("s", "k"); v != "a\nb\nc" {
		t.Fatalf(`Val=%q want "a\nb\nc"`, v)
	}
	// PushVal to a non-existent parameter fails.
	if err := cfg.PushVal("s", "missing", "x"); err == nil {
		t.Fatalf("PushVal on missing param should fail")
	}
	// Empty value case is a no-op success on existing param.
	if err := cfg.PushVal("s", "k"); err != nil {
		t.Fatalf("empty PushVal should be no-op: %v", err)
	}
	// Output is here-doc form (multi-valued).
	var buf bytes.Buffer
	cfg.WriteTo(&buf)
	if !strings.Contains(buf.String(), "k= <<") {
		t.Fatalf("expected here-doc output for multi value, got:\n%s", buf.String())
	}
}

// Exercises SetParameterEOT / GetParameterEOT / DeleteParameterEOT.
func TestEOTAccessors(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "x", "y")
	// A multi-valued param auto-gets the default "EOT" marker.
	if got := cfg.GetParameterEOT("s", "k"); got != "EOT" {
		t.Fatalf("GetParameterEOT=%q want \"EOT\"", got)
	}
	if err := cfg.SetParameterEOT("s", "k", "EOF"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetParameterEOT("s", "k"); got != "EOF" {
		t.Fatalf("GetParameterEOT=%q want \"EOF\"", got)
	}
	var buf bytes.Buffer
	cfg.WriteTo(&buf)
	if !strings.Contains(buf.String(), "k= <<EOF") {
		t.Fatalf("expected here-doc with EOF marker, got:\n%s", buf.String())
	}
	// After deleting the explicit EOT, the marker reverts to the default "EOT"
	// (a multi-valued param keeps here-doc output).
	if err := cfg.DeleteParameterEOT("s", "k"); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetParameterEOT("s", "k"); got != "EOT" {
		t.Fatalf("GetParameterEOT after delete=%q want \"EOT\"", got)
	}
}

// Exercises RemoveGroupMember via DeleteSection.
func TestRemoveGroupMember(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatal(err)
	}
	before := cfg.GroupMembers("group")
	if len(before) != 3 {
		t.Fatalf("group members before=%v want 3", before)
	}
	if err := cfg.DeleteSection("group member one"); err != nil {
		t.Fatal(err)
	}
	after := cfg.GroupMembers("group")
	for _, m := range after {
		if m == "group member one" {
			t.Fatalf("removed member still present: %v", after)
		}
	}
	if len(after) != 2 {
		t.Fatalf("group members after=%v want 2", after)
	}
	// Public RemoveGroupMember no-op on a non-grouped section.
	cfg.RemoveGroupMember("test1")
}

// Exercises CopySection/RenameSection with the group-members option.
func TestRenameSectionWithGroup(t *testing.T) {
	cfg := New()
	cfg.AddSection("g one")
	cfg.AddSection("g two")
	cfg.NewVal("g one", "k", "v")
	if err := cfg.RenameSection("g one", "g one renamed", false); err != nil {
		t.Fatal(err)
	}
	if cfg.SectionExists("g one") {
		t.Fatalf("old section should be gone")
	}
	if !cfg.SectionExists("g one renamed") {
		t.Fatalf("renamed section should exist")
	}
	if v, _ := cfg.Val("g one renamed", "k"); v != "v" {
		t.Fatalf(`renamed Val=%q want "v"`, v)
	}
}

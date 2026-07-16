package inifiles

import (
	"testing"
)

// CopySection with includeGroup=true recursively copies group-member
// counterparts. The port's GroupMembers keys on the GROUP name, while
// CopySection passes the section name; so the loop only iterates when the
// section name coincides with a group key. Using a single-word section "g"
// (added directly) alongside grouped sections "g one"/"g two" exercises the
// recursive copy and copySectionPrefix (currently 0% / 66% covered).
func TestCopySection_IncludeGroup(t *testing.T) {
	cfg := New()
	cfg.AddSection("g")           // single-word section == group key "g"
	cfg.NewVal("g one", "k", "1") // enrolls group "g"
	cfg.NewVal("g two", "k", "2")

	if err := cfg.CopySection("g", "X", true); err != nil {
		t.Fatalf("CopySection includeGroup: %v", err)
	}
	v, ok := cfg.Val("X one", "k")
	if !ok || v != "1" {
		t.Errorf("Val(X one,k) = (%q,%v), want (1,true)", v, ok)
	}
	v, ok = cfg.Val("X two", "k")
	if !ok || v != "2" {
		t.Errorf("Val(X two,k) = (%q,%v), want (2,true)", v, ok)
	}
	// Originals are still present.
	if _, ok := cfg.Val("g one", "k"); !ok {
		t.Errorf("original g one removed by copy")
	}
	if !cfg.SectionExists("g") {
		t.Errorf("original single-word section g removed by copy")
	}
}

// copySectionPrefix unit cases (currently 0%).
func TestCopySectionPrefix(t *testing.T) {
	cases := []struct {
		member, old, newName, want string
	}{
		{"grp one", "grp", "grpX", "grpX one"}, // standard prefix replace
		{"group one", "grp", "X", "group one"}, // member does not start with old
		{"grp", "grp", "grpX", "grpX"},         // exact match
		{"grpprefix", "grp", "X", "Xprefix"},   // len(member)>=len(old), starts with old
	}
	for _, tc := range cases {
		got := copySectionPrefix(tc.member, tc.old, tc.newName)
		if got != tc.want {
			t.Errorf("copySectionPrefix(%q,%q,%q) = %q, want %q", tc.member, tc.old, tc.newName, got, tc.want)
		}
	}
}

// CopySection is a no-op when old/newName is empty, old doesn't exist, or the
// destination already exists (covers the early-return branches).
func TestCopySection_NoopCases(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "k", "v")

	if err := cfg.CopySection("", "T", false); err != nil {
		t.Errorf("CopySection(\"\",...) = %v, want nil", err)
	}
	if err := cfg.CopySection("S", "", false); err != nil {
		t.Errorf("CopySection(...,\"\",...) = %v, want nil", err)
	}
	if err := cfg.CopySection("NoSect", "T", false); err != nil {
		t.Errorf("CopySection(NoSect,...) = %v, want nil", err)
	}
	// Destination exists: no-op (no overwrite).
	if err := cfg.CopySection("S", "S", false); err != nil {
		t.Errorf("CopySection(S,S) = %v, want nil", err)
	}
}

// RenameSection copies then deletes old, dropping the original. Using a
// single-word section "g" matching the group key "g" lets the group members be
// copied to "X one"/"X two" before "g" is deleted.
func TestRenameSection(t *testing.T) {
	cfg := New()
	cfg.AddSection("g")
	cfg.NewVal("g one", "k", "1")
	cfg.NewVal("g two", "k", "2")

	if err := cfg.RenameSection("g", "X", true); err != nil {
		t.Fatalf("RenameSection: %v", err)
	}
	if cfg.SectionExists("g") {
		t.Errorf("old section g should have been deleted")
	}
	if v, ok := cfg.Val("X one", "k"); !ok || v != "1" {
		t.Errorf("Val(X one,k) = (%q,%v), want (1,true)", v, ok)
	}
	if v, ok := cfg.Val("X two", "k"); !ok || v != "2" {
		t.Errorf("Val(X two,k) = (%q,%v), want (2,true)", v, ok)
	}
	// "g one"/"g two" stay (only old==g is deleted by rename).
	if _, ok := cfg.Val("g one", "k"); !ok {
		t.Errorf("Val(g one,k) gone, should still exist")
	}
}

// RenameSection copying to a missing source returns nil (via CopySection).
func TestRenameSection_MissingSource(t *testing.T) {
	cfg := New()
	if err := cfg.RenameSection("NoSect", "T", false); err != nil {
		t.Errorf("RenameSection(NoSect,...) = %v, want nil", err)
	}
}

// SectionExists returns false for the empty string.
func TestSectionExists_Empty(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "k", "v")
	if cfg.SectionExists("") {
		t.Errorf("SectionExists(\"\") = true, want false")
	}
	if !cfg.SectionExists("S") {
		t.Errorf("SectionExists(S) = false, want true")
	}
	if cfg.SectionExists("NoSect") {
		t.Errorf("SectionExists(NoSect) = true, want false")
	}
}

// sectionOf returns nil for the empty string (covers the empty-name branch).
func TestSectionOf_Empty(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "k", "v")
	if s := cfg.sectionOf(""); s != nil {
		t.Errorf("sectionOf(\"\") = %v, want nil", s)
	}
}

// Delete wipes every section and resets groups/groupOrder.
func TestDelete_ClearsAll(t *testing.T) {
	cfg := New()
	cfg.NewVal("grp one", "k", "1")
	cfg.NewVal("S", "k", "v")

	cfg.Delete()
	if got := cfg.Sections(); len(got) != 0 {
		t.Errorf("Sections() after Delete = %v, want empty", got)
	}
	if got := cfg.Groups(); len(got) != 0 {
		t.Errorf("Groups() after Delete = %v, want empty", got)
	}
	if cfg.SectionExists("S") {
		t.Errorf("SectionExists(S) = true after Delete")
	}
}

// AddSection is a no-op for the empty string and for an existing section.
func TestAddSection_NoopCases(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "k", "v")

	if err := cfg.AddSection(""); err != nil {
		t.Errorf("AddSection(\"\") = %v, want nil", err)
	}
	if err := cfg.AddSection("S"); err != nil {
		t.Errorf("AddSection(S) existing = %v, want nil", err)
	}
	// Idempotent: still exactly one S section.
	n := 0
	for _, name := range cfg.Sections() {
		if name == "S" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("after duplicate AddSection, S count = %d, want 1", n)
	}
}

// DeleteSection on a missing section is a no-op.
func TestDeleteSection_Missing(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteSection("NoSect"); err != nil {
		t.Errorf("DeleteSection(NoSect) = %v, want nil", err)
	}
}

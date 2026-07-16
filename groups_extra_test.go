package inifiles

import (
	"testing"
)

// SetGroupMember is a public delegate for setGroupMember; ensure it enrolls and
// dedups without panicking on names lacking a second word.
func TestSetGroupMember_EnrollAndDedup(t *testing.T) {
	cfg := New()
	cfg.AddSection("grp one")
	cfg.AddSection("grp two")

	// Explicit (redundant) enrollment should not duplicate the member.
	cfg.SetGroupMember("grp one")
	cfg.SetGroupMember("grp one")

	members := cfg.GroupMembers("grp")
	if len(members) != 2 || members[0] != "grp one" || members[1] != "grp two" {
		t.Fatalf("GroupMembers(grp) = %v, want [grp one grp two]", members)
	}
}

// A section name with no second word yields no group (groupFromName -> false).
func TestSetGroupMember_NoSecondWord(t *testing.T) {
	cfg := New()
	cfg.AddSection("lonely")
	cfg.SetGroupMember("lonely")
	if g := cfg.GroupMembers("lonely"); g != nil {
		t.Errorf("GroupMembers(lonely) = %v, want nil", g)
	}
	if got := cfg.Groups(); len(got) != 0 {
		t.Errorf("Groups() = %v, want empty", got)
	}
}

// RemoveGroupMember drops the member, and when the group empties it is removed
// from both the map and groupOrder (covers the cleanup-to-empty branch).
func TestRemoveGroupMember_EmptiesAndRemovesGroup(t *testing.T) {
	cfg := New()
	cfg.AddSection("grp only")

	if g := cfg.GroupMembers("grp"); len(g) != 1 || g[0] != "grp only" {
		t.Fatalf("GroupMembers(grp) = %v, want [grp only]", g)
	}

	cfg.RemoveGroupMember("grp only")

	if g := cfg.GroupMembers("grp"); g != nil {
		t.Errorf("GroupMembers(grp) after remove = %v, want nil", g)
	}
	if got := cfg.Groups(); len(got) != 0 {
		t.Errorf("Groups() after remove = %v, want empty (groupOrder cleanup)", got)
	}
}

// removeGroupMember for a name that has no group (no second word) is a no-op.
func TestRemoveGroupMember_NoGroup(t *testing.T) {
	cfg := New()
	cfg.AddSection("lonely")
	cfg.RemoveGroupMember("lonely") // must not panic
	if got := cfg.Groups(); len(got) != 0 {
		t.Errorf("Groups() = %v, want empty", got)
	}
}

// Removing a non-member of an existing group leaves the others intact.
func TestRemoveGroupMember_KeepsOthers(t *testing.T) {
	cfg := New()
	cfg.AddSection("grp one")
	cfg.AddSection("grp two")
	cfg.RemoveGroupMember("grp one")

	members := cfg.GroupMembers("grp")
	if len(members) != 1 || members[0] != "grp two" {
		t.Fatalf("GroupMembers(grp) = %v, want [grp two]", members)
	}
	if got := cfg.Groups(); len(got) != 1 || got[0] != "grp" {
		t.Errorf("Groups() = %v, want [grp]", got)
	}
}

// GroupMembers returns nil for the empty string and for unknown groups.
func TestGroupMembers_EmptyAndUnknown(t *testing.T) {
	cfg := New()
	cfg.AddSection("grp one")

	if g := cfg.GroupMembers(""); g != nil {
		t.Errorf("GroupMembers(\"\") = %v, want nil", g)
	}
	if g := cfg.GroupMembers("nope"); g != nil {
		t.Errorf("GroupMembers(nope) = %v, want nil", g)
	}
}

// Under NoCase, group keys are lowercased and members dedup case-insensitively
// when enrolled via AddSection with differing case.
func TestGroups_NoCaseKeying(t *testing.T) {
	cfg := New(WithNoCase(true))
	cfg.AddSection("Grp one")
	cfg.AddSection("GRP two")

	members := cfg.GroupMembers("grp")
	if len(members) != 2 {
		t.Fatalf("GroupMembers(grp) nocase = %v, want 2 members", members)
	}
	got := cfg.Groups()
	if len(got) != 1 || got[0] != "grp" {
		t.Errorf("Groups() nocase = %v, want [grp]", got)
	}

	// Remove empties the nocase-keyed group and cleans groupOrder.
	cfg.RemoveGroupMember("grp one")
	cfg.RemoveGroupMember("grp two")
	if got := cfg.Groups(); len(got) != 0 {
		t.Errorf("Groups() after nocase cleanup = %v, want empty", got)
	}
}

// groupFromName boundary conditions: leading whitespace, trailing-only word,
// single word, empty.
func TestGroupFromName_Boundaries(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
		want string
	}{
		{"", false, ""},
		{"word", false, ""},
		{" word", false, ""},  // i<=0: leading whitespace at index 0 -> i==0
		{"word  ", false, ""}, // trailing whitespace only, no second word
		{"word\t", false, ""},
		{"a b", true, "a"},
		{"grp one two", true, "grp"},
	}
	for _, tc := range cases {
		g, ok := groupFromName(tc.name)
		if ok != tc.ok {
			t.Errorf("groupFromName(%q) ok = %v, want %v", tc.name, ok, tc.ok)
			continue
		}
		if ok && g != tc.want {
			t.Errorf("groupFromName(%q) = %q, want %q", tc.name, g, tc.want)
		}
	}
}

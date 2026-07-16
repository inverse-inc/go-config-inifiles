package inifiles

import (
	"testing"
)

// translated from config-inifiles/t/08group.t
func TestGroupMembersWithSpaces(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	want := []string{"group member one", "group member two", "group member three"}
	got := cfg.GroupMembers("group")
	if len(got) != len(want) {
		t.Fatalf("GroupMembers(group) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GroupMembers(group)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

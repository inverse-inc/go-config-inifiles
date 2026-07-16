package inifiles

import "testing"

// translated from config-inifiles/t/18non-contiguous-groups.t
//
// GroupMembers returns all sections sharing a group prefix in the order they
// appear in the file, even when members of other groups are interleaved
// between them.
func TestNonContiguousGroups_OrderPreserved(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "non-contiguous-groups.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	got := cfg.GroupMembers("A")
	want := []string{"A 1", "A 2", "A 3"}
	if len(got) != len(want) {
		t.Fatalf("GroupMembers(A) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GroupMembers(A)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

package inifiles

import (
	"testing"
)

// Translated from config-inifiles/t/20allowedcommentchars.t.
func TestAllowedCommentChars(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "allowed-comment-chars.ini"),
		WithAllowedCommentChars([]rune{'}', ';'}))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	// Proper comments (lines starting with '}' or ';') are ignored; the
	// real mykey = 500 is loaded, not the commented-out 1000.
	if v, ok := cfg.Val("cat1", "mykey"); !ok || v != "500" {
		t.Errorf(`Val(cat1,mykey) = %q ok=%v, want "500"`, v, ok)
	}
}

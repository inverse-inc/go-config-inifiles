package inifiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rewrite is a no-op (returns nil) when no file is configured.
func TestRewrite_NoFile(t *testing.T) {
	cfg := New()
	if err := cfg.Rewrite(); err != nil {
		t.Errorf("Rewrite no file = %v, want nil", err)
	}
}

// WriteFile refuses to write to a read-only destination (the Stat-ok /
// OpenFile-fail branch of writeFileTo). Skipped under root, where permission
// checks are bypassed.
func TestWriteFile_ReadOnlyDestination(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; read-only permission check is bypassed")
	}
	dir := t.TempDir()
	roPath := filepath.Join(dir, "ro.ini")
	if err := os.WriteFile(roPath, []byte("[S]\nk=v\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roPath, 0444); err != nil {
		t.Fatal(err)
	}

	cfg := New()
	cfg.NewVal("S", "k", "v")
	cfg.SetFileName(roPath)
	if err := cfg.WriteFile(roPath); err == nil {
		t.Errorf("WriteFile to read-only destination = nil, want error")
	}
	// Destination contents unchanged.
	data, err := os.ReadFile(roPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "[S]\nk=v" {
		t.Errorf("read-only destination modified; contents=\n%s", string(data))
	}
}

// WriteFile errors when the target directory does not exist (CreateTemp in
// writeFileTo fails).
func TestWriteFile_NonexistentDir(t *testing.T) {
	cfg := New()
	cfg.NewVal("S", "k", "v")
	missing := filepath.Join(t.TempDir(), "no-such-subdir", "c.ini")
	if err := cfg.WriteFile(missing); err == nil {
		t.Errorf("WriteFile to nonexistent dir = nil, want error")
	}
}

// ensureDirStat records the existing target's mode when no explicit mode was
// set; a subsequent WriteFile preserves that mode.
func TestWriteFile_PreservesExistingMode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; mode preservation check is unreliable")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "c.ini")
	if err := os.WriteFile(p, []byte("[Old]\nk=v\n"), 0640); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg.NewVal("S", "k", "v")
	if err := cfg.WriteFile(p); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("perm after WriteFile = %o, want 0640", info.Mode().Perm())
	}
}

// WriteFile with an explicit SetWriteMode applies that mode via chmod.
func TestWriteFile_AppliesExplicitMode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.ini")
	cfg := New()
	cfg.NewVal("S", "k", "v")
	cfg.SetWriteMode(0666)
	if err := cfg.WriteFile(p); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0666 {
		t.Errorf("perm after WriteFile = %o, want 0666", info.Mode().Perm())
	}
}

// Rewrite writes back to the configured file.
func TestRewrite_WritesBack(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.ini")
	if err := os.WriteFile(p, []byte("[S]\nk=v\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg.NewVal("S", "k2", "v2")
	if err := cfg.Rewrite(); err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	reloaded, err := NewFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := reloaded.Val("S", "k2"); !ok || v != "v2" {
		t.Errorf("reloaded Val(S,k2) = (%q,%v), want (v2,true)", v, ok)
	}
}

// orRunes returns {newline} when no line endings detected, else the rune set
// of the detected endings (deduplicated in first-seen order).
func TestOrRunes(t *testing.T) {
	cfg := New()
	got := cfg.orRunes()
	if len(got) != 1 || got[0] != '\n' {
		t.Errorf("orRunes empty = %v, want [\\n]", got)
	}
	cfg.lineEnds = []byte{'\r', '\n'}
	got = cfg.orRunes()
	if len(got) != 2 || got[0] != '\r' || got[1] != '\n' {
		t.Errorf("orRunes [\\r \\n] = %v, want [\\r \\n]", got)
	}
	// Dedup duplicate bytes.
	cfg.lineEnds = []byte{'\n', '\n', '\r'}
	got = cfg.orRunes()
	if len(got) != 2 || got[0] != '\n' || got[1] != '\r' {
		t.Errorf("orRunes dedup = %v, want [\\n \\r]", got)
	}
}

// ors returns the detected line ending as a string, or "\n".
func TestOrs(t *testing.T) {
	cfg := New()
	if got := cfg.ors(); got != "\n" {
		t.Errorf("ors empty = %q, want \\n", got)
	}
	cfg.lineEnds = []byte{'\r', '\n'}
	if got := cfg.ors(); got != "\r\n" {
		t.Errorf("ors = %q, want \\r\\n", got)
	}
}

// splitOnOrsRunes splits on any of the given runes; empty input -> [""];
// runes >= 128 are ignored (only ASCII bytes are admitted to the set).
func TestSplitOnOrsRunes(t *testing.T) {
	if got := splitOnOrsRunes("", []rune{'\n'}); len(got) != 1 || got[0] != "" {
		t.Errorf("splitOnOrsRunes(\"\") = %v, want [\"\"]", got)
	}
	got := splitOnOrsRunes("a\nb\rc", []rune{'\n', '\r'})
	if !sliceEq(got, []string{"a", "b", "c"}) {
		t.Errorf("splitOnOrsRunes = %v, want [a b c]", got)
	}
	// Non-ASCII rune ignored (won't split on it).
	got = splitOnOrsRunes("x✓y", []rune{'✓'})
	if !sliceEq(got, []string{"x✓y"}) {
		t.Errorf("splitOnOrsRunes non-ascii = %v, want [x✓y]", got)
	}
}

// ensureEndComment returns "" for empty, else " <char> <cmt>".
func TestEnsureEndComment(t *testing.T) {
	cfg := New() // CommentChar default '#'
	if got := cfg.ensureEndComment(""); got != "" {
		t.Errorf("ensureEndComment(\"\") = %q, want empty", got)
	}
	if got := cfg.ensureEndComment("hi"); got != " # hi" {
		t.Errorf("ensureEndComment(hi) = %q, want ' # hi'", got)
	}
}

// WriteFile surfaces an error when the atomic rename of the temp file to the
// destination fails (writeFileTo rename branch). A NUL byte in the basename is
// rejected by the filesystem while leaving the directory component valid, so
// CreateTemp succeeds and the failure surfaces from os.Rename.
func TestWriteFile_RenameError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "\x00bad-name")
	cfg := New()
	if err := cfg.NewVal("S", "k", "v"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.WriteFile(badPath); err == nil {
		t.Errorf("WriteFile with NUL in basename = nil, want error")
	}
	// The temp file must have been cleaned up by the rename-error branch.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "temp.ini-") {
			t.Errorf("leftover temp file after rename error: %s", e.Name())
		}
	}
}

// calcEOTMark grows the marker when the value contains the current mark.
func TestCalcEOTMark(t *testing.T) {
	cfg := New()
	p := &param{name: "x"} // hasEOT false -> default "EOT"
	if got := cfg.calcEOTMark(p, []string{"some EOT here"}); got != "EOTA" {
		t.Errorf("calcEOTMark with EOT in value = %q, want EOTA", got)
	}
	p2 := &param{name: "x", hasEOT: true, eot: "M"}
	if got := cfg.calcEOTMark(p2, []string{"no marker here"}); got != "M" {
		t.Errorf("calcEOTMark explicit M = %q, want M", got)
	}
	if got := cfg.calcEOTMark(p2, []string{"has M"}); got != "MA" {
		t.Errorf("calcEOTMark explicit M collision = %q, want MA", got)
	}
}

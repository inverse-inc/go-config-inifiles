package inifiles

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errReader is an io.Reader that always fails (to exercise readFromReader's
// error branch).
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// NewFromReader surfaces read errors from the underlying reader.
func TestNewFromReader_ReadError(t *testing.T) {
	if _, err := NewFromReader(errReader{}); err == nil {
		t.Errorf("NewFromReader(errReader) err=nil, want error")
	}
}

// SetFileName("") returns ErrEmptyArg and leaves the previously set name in
// place (covers the empty-name early return).
func TestSetFileName_EmptyNoop(t *testing.T) {
	cfg := New()
	if err := cfg.SetFileName("/some/path"); err != nil {
		t.Fatalf("SetFileName(/some/path): %v", err)
	}
	if err := cfg.SetFileName(""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetFileName(\"\") = %v, want ErrEmptyArg", err)
	}
	if got := cfg.GetFileName(); got != "/some/path" {
		t.Errorf("GetFileName() = %q after empty SetFileName, want /some/path", got)
	}
}

// GetWriteMode returns "" when no mode has been set (covers the fileMode==0
// branch), and the formatted octal string after a valid SetWriteMode.
func TestGetWriteMode_NoModeAndRoundTrip(t *testing.T) {
	cfg := New()
	if got := cfg.GetWriteMode(); got != "" {
		t.Errorf("GetWriteMode() = %q, want \"\" (no mode set)", got)
	}
	if err := cfg.SetWriteMode("644"); err != nil {
		t.Fatalf("SetWriteMode(644): %v", err)
	}
	if got := cfg.GetWriteMode(); got != "0644" {
		t.Errorf("GetWriteMode() = %q, want 0644", got)
	}
}

// SetWriteMode rejects non-octal digits and too-short strings.
func TestSetWriteMode_Invalid(t *testing.T) {
	cfg := New()
	for _, m := range []string{"", "6", "64", "89a", "abc", "7777x"} {
		if err := cfg.SetWriteMode(m); err == nil {
			t.Errorf("SetWriteMode(%q) = nil, want error", m)
		}
	}
}

// isValidOctalMode / parseOctalMode / formatOctalMode direct unit tests.
func TestOctalModeHelpers(t *testing.T) {
	if !isValidOctalMode("644") {
		t.Errorf("isValidOctalMode(644) = false, want true")
	}
	if !isValidOctalMode("0666") {
		t.Errorf("isValidOctalMode(0666) = false, want true")
	}
	if isValidOctalMode("8") || isValidOctalMode("9ab") || isValidOctalMode("ab") {
		t.Errorf("isValidOctalMode rejected 8/9ab/ab incorrectly true")
	}
	if got := formatOctalMode(parseOctalMode("0644")); got != "0644" {
		t.Errorf("formatOctalMode(parseOctalMode(0644)) = %q, want 0644", got)
	}
	if got := formatOctalMode(parseOctalMode("0666")); got != "0666" {
		t.Errorf("formatOctalMode(parseOctalMode(0666)) = %q, want 0666", got)
	}
}

// Reload on a configured-but-missing file surfaces os.Open's error.
func TestReload_OpenError(t *testing.T) {
	cfg := New()
	cfg.SetFileName(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err := cfg.Reload(); err == nil {
		t.Errorf("Reload on missing file returned nil, want error")
	}
}

// Reload with ReloadWarn writes a reload notice (to a custom ReloadWriter on the
// second reload) — covers writeReloadWarn, WithReloadWarn and WithReloadWriter.
func TestReload_ReloadWarnWrites(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.ini")
	if err := os.WriteFile(cfgPath, []byte("[S]\nk=v\n"), 0644); err != nil {
		t.Fatal(err)
	}
	warn, err := os.CreateTemp(dir, "warn-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := NewFromFile(cfgPath, WithReloadWarn(true), WithReloadWriter(warn))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	// Second reload: reloadCnt goes 2 > 1, triggering writeReloadWarn into warn.
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if err := warn.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := warn.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(warn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "reloading config file") {
		t.Errorf("reload warning output = %q, want substring \"reloading config file\"", string(data))
	}
}

// ReloadWarn defaults to os.Stderr when no ReloadWriter is configured (covers
// the w == nil fallback branch in writeReloadWarn). We only assert Reload
// succeeds without panicking, since redirecting stderr reliably is fiddly.
func TestReload_ReloadWarnDefaultWriter(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.ini")
	if err := os.WriteFile(cfgPath, []byte("[S]\nk=v\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(cfgPath, WithReloadWarn(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
}

// Errors() returns nil when no parse errors accumulated.
func TestConfigErrors_None(t *testing.T) {
	cfg := New()
	if got := cfg.Errors(); got != nil {
		t.Errorf("Errors() on clean config = %v, want nil", got)
	}
}

// applyImport with no import configured is a no-op (early return); Reload on a
// file-less config deep-copies nothing and succeeds.
func TestReload_NoFileNoImport(t *testing.T) {
	cfg := New()
	if err := cfg.Reload(); err != nil {
		t.Errorf("Reload no-file: %v", err)
	}
	if got := cfg.Sections(); len(got) != 0 {
		t.Errorf("Sections() = %v, want empty", got)
	}
}

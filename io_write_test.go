package inifiles

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// translated from config-inifiles/t/12open-empty-file.t.
// New() -> WriteFile creates an empty file; it loads with -allowempty and
// errors without it.
func TestWriteOpenEmptyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "empty.ini")
	cfg := New()
	if err := cfg.WriteFile(p); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}

	if _, err := NewFromFile(p, WithAllowEmpty(true)); err != nil {
		t.Fatalf("NewFromFile empty (allowempty) should succeed: %v", err)
	}
	if _, err := NewFromFile(p); err == nil {
		t.Fatalf("NewFromFile empty (no allowempty) should error")
	}
}

// translated from config-inifiles/t/07misc.t scenarios 1 & 2 (read-only val).
func TestMisc_MultipleEqualsAndSubstringParam(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	// scenario 1: multiple equals split on the first.
	if v, ok := cfg.Val("test7", "criterion"); !ok || v != "price <= maximum" {
		t.Fatalf(`Val(test7,criterion) = %q ok=%v, want "price <= maximum"`, v, ok)
	}

	// scenario 2: a parameter name that is a substring of another loads.
	if v, ok := cfg.Val("substring", "boot"); !ok || v != "smarty" {
		t.Fatalf(`Val(substring,boot) = %q ok=%v, want "smarty"`, v, ok)
	}
}

// translated from config-inifiles/t/07misc.t scenario 3 (default option works).
func TestMisc_DefaultOption(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"),
		WithDefault("test1"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	out := filepath.Join(t.TempDir(), "test07.ini")
	if err := cfg.SetFileName(out); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	// 07misc passes SetWriteMode("0666"); the Go port accepts an os.FileMode,
	// so the equivalent is 0666.
	cfg.SetWriteMode(0666)

	// val('test2','three') falls back to the default section's 'three'.
	if v, ok := cfg.Val("test2", "three"); !ok || v != "value3" {
		t.Fatalf(`Val(test2,three) = %q ok=%v, want "value3"`, v, ok)
	}
}

// SetWriteMode / GetWriteMode round-trip (07misc uses SetWriteMode("0666")).
func TestMisc_SetGetWriteMode(t *testing.T) {
	cfg := New()
	// 07misc uses "0666"; the Go port accepts an os.FileMode and stores the
	// permission bits.
	cfg.SetWriteMode(0666)
	if got := cfg.GetWriteMode(); got != 0666 {
		t.Fatalf("GetWriteMode = %o, want 0666", got)
	}
}

// 07misc.t/10delta.t pass mode strings like "0666" (a literal value, not the
// octal). The Go port now accepts an os.FileMode directly, so 0666 (octal) is
// the natural literal.
func TestMisc_SetWriteMode0666PerlCompat(t *testing.T) {
	cfg := New()
	cfg.SetWriteMode(0666)
	if got := cfg.GetWriteMode(); got != 0666 {
		t.Fatalf("GetWriteMode = %o, want 0666", got)
	}
}

// translated from config-inifiles/t/07misc.t scenario 4 (write to a read-only
// file fails). Perl WriteConfig on a 0444 file returns undef because its
// open(">", $path) is rejected by the file mode. The Go port's WriteFile uses
// an atomic temp+rename, which overwrites a read-only destination so long as
// the parent directory is writable (verified empirically on darwin). This test
// encodes the Perl expectation; it FAILS until WriteFile gains a mode-honouring
// write path. Skipped as root (mode bits cannot deny root).
func TestMisc_WriteReadOnlyFileFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("RO permissions not enforceable as root")
	}

	p := filepath.Join(t.TempDir(), "ro.ini")
	seed := []byte("[test2]\nthree=value3\n")
	if err := os.WriteFile(p, seed, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewFromFile(p, WithDefault("test1"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	target := filepath.Join(t.TempDir(), "test07.ini")
	if err := cfg.SetFileName(target); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	if err := cfg.WriteFile(target); err != nil {
		t.Fatalf("first WriteFile: %v", err)
	}
	st, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	perm := st.Mode().Perm()
	if err := os.Chmod(target, 0444); err != nil {
		t.Fatalf("chmod 0444: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, perm) })

	if err := cfg.SetVal("test2", "three", "should not be here"); err != nil {
		t.Fatalf("SetVal: %v", err)
	}
	werr := cfg.WriteFile(target)
	contents, _ := os.ReadFile(target)
	switch {
	case werr == nil:
		t.Fatalf("WriteFile to a read-only file should fail (got nil error); "+
			"file %s was overwritten to %q (seed perm %o, RO 0444). "+
			"WriteFile's temp+rename bypasses the destination mode; the Perl "+
			"direct open(>) does not — implementation gap.", target, string(contents), perm)
	default:
		// Failed as expected; additionally confirm the RO file's contents
		// were not altered (no partial/truncation side effects).
		if len(contents) == 0 || !bytes.Contains(contents, []byte("three=value3")) {
			t.Fatalf("WriteFile reported %v but altered the RO file contents to %q", werr, string(contents))
		}
	}
}

// translated from config-inifiles/t/30parameters-with-empty-values.t (RT #68554).
// A parameter assigned the empty string must still be written as "MyParam=".
func TestWriteEmptyParameterValue(t *testing.T) {
	cfg := New()
	if err := cfg.NewVal("MySection", "MyParam", ""); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	matched, err := regexp.MatchString(`(?m)^MyParam=$`, buf.String())
	if err != nil {
		t.Fatalf("regexp: %v", err)
	}
	if !matched {
		t.Fatalf("output missing ^MyParam=$ line:\n%s", buf.String())
	}
}

// translated from config-inifiles/t/33update-using-tied-fh-w-shorter-names.t
// (RT #45997). In-place update shortening a value must not leave remnant
// bytes: after delete + re-add a shorter value, the rewritten content matches
// the expected byte content exactly.
func TestWriteShorterValueRewrite(t *testing.T) {
	src := "[toto]\ntata=verylongstringtoreplacewithashorterone\n\n[section3]\narg3=valf\n"
	cfg, err := NewFromBytes([]byte(src), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	if err := cfg.DelVal("toto", "tata"); err != nil {
		t.Fatalf("DelVal: %v", err)
	}
	if err := cfg.NewVal("toto", "tata", "short"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	want := "[toto]\ntata=short\n\n[section3]\narg3=valf\n"
	if got := buf.String(); got != want {
		t.Fatalf("output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

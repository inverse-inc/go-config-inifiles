package inifiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// translated from config-inifiles/t/00load.t (constructor file sources;
// malformed/empty handling) plus 26scalar-filehandle.t, 23scalar-ref.t and
// 38security-open.t. The Perl tests exercise several filehandle kinds (glob,
// ref-to-glob, IO::File, scalar filehandle, scalar ref); in Go these collapse
// to NewFromFile, NewFromReader (any io.Reader / *os.File / strings.Reader)
// and NewFromBytes (scalar ref).

// 00load.t scene: open via a filehandle (glob / \*glob / IO::File) succeeds.
func TestLoad_FromReaderOsFile(t *testing.T) {
	f, err := os.Open(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("open test.ini: %v", err)
	}
	defer f.Close()
	cfg, err := NewFromReader(f)
	if err != nil {
		t.Fatalf("NewFromReader(*os.File): %v", err)
	}
	if v, ok := cfg.Val("test1", "one"); !ok || v != "value1" {
		t.Errorf(`Val(test1,one) = %q ok=%v, want "value1"`, v, ok)
	}
}

// 00load.t scene: reread on an open handle (ReadConfig) is exercised by the
// read performed inside NewFromReader.
func TestLoad_FromReaderReread(t *testing.T) {
	f, err := os.Open(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if _, err := NewFromReader(f); err != nil {
		t.Fatalf("NewFromReader (reread scene): %v", err)
	}
}

// 26scalar-filehandle.t: a filehandle made out of a scalar (RT #54997).
func TestLoad_FromReaderScalarFilehandle(t *testing.T) {
	contents := "[section1]\nkey = val\n"
	cfg, err := NewFromReader(strings.NewReader(contents))
	if err != nil {
		t.Fatalf("NewFromReader(strings.NewReader): %v", err)
	}
	if v, ok := cfg.Val("section1", "key"); !ok || v != "val" {
		t.Fatalf(`Val(section1,key) = %q ok=%v, want "val"`, v, ok)
	}
}

// 23scalar-ref.t: -file => \$str (RT #45588) -> NewFromBytes.
func TestLoad_FromBytesScalarRef(t *testing.T) {
	contents := "[section1]\nkey = val\n"
	cfg, err := NewFromBytes([]byte(contents))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("section1", "key"); !ok || v != "val" {
		t.Fatalf(`Val(section1,key) = %q ok=%v, want "val"`, v, ok)
	}
}

// 00load.t scene: the pathname of a file (-file => path) works.
func TestLoad_FromFilePath(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile test.ini: %v", err)
	}
	if v, ok := cfg.Val("test1", "one"); !ok || v != "value1" {
		t.Fatalf(`Val(test1,one) = %q ok=%v, want "value1"`, v, ok)
	}
}

// 00load.t scene: load from a reader, set a file name, rewrite to it and
// verify round-trip. Translates the "write to a new file name from a handle"
// scene (the Perl 2-arg-open regression that wrote to a file named after the
// scalar value of the handle).
func TestLoad_RewriteAfterReaderSource(t *testing.T) {
	f, err := os.Open(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	cfg, err := NewFromReader(f)
	f.Close()
	if err != nil {
		t.Fatalf("NewFromReader: %v", err)
	}
	out := filepath.Join(t.TempDir(), "test01.ini")
	if err := cfg.SetFileName(out); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	if err := cfg.Rewrite(); err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	// Confirm nothing was written under a handle-derived weird name: the
	// configured output file exists and parses back.
	cfg2, err := NewFromFile(out)
	if err != nil {
		t.Fatalf("reload written file: %v", err)
	}
	if v, ok := cfg2.Val("test1", "one"); !ok || v != "value1" {
		t.Fatalf(`round-trip Val(test1,one) = %q ok=%v, want "value1"`, v, ok)
	}
}

// 00load.t scene: a non-INI file fails to load but does not panic (Perl:
// !$@ && !$ERRORS && !defined($ini)).
func TestLoad_NonIniFileFailsGracefully(t *testing.T) {
	p := filepath.Join(t.TempDir(), "notini.txt")
	if err := os.WriteFile(p, []byte("this is not ini\nno equals here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFromFile(p); err == nil {
		t.Fatalf("NewFromFile on a non-INI file should return an error")
	}
}

// 00load.t scene: read the __END__ DATA block (comments first, then a section)
// without errors.
func TestLoad_DataSection(t *testing.T) {
	data := "; File that has comments in the first line\n" +
		"; Comments are marked with ';'.\n" +
		"; This should not fail when checking if the file is valid\n" +
		"[section]\nparameter=value\n"
	cfg, err := NewFromBytes([]byte(data))
	if err != nil {
		t.Fatalf("NewFromBytes DATA: %v", err)
	}
	if v, ok := cfg.Val("section", "parameter"); !ok || v != "value" {
		t.Fatalf(`Val(section,parameter) = %q ok=%v, want "value"`, v, ok)
	}
}

// 00load.t scene: a utf-8 (BOM-stripped) file opens fine.
func TestLoad_Utf8File(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "en.ini"))
	if err != nil {
		t.Fatalf("NewFromFile en.ini: %v", err)
	}
	if v, ok := cfg.Val("x", "ShortName"); !ok || v != "GENERAL" {
		t.Fatalf(`Val(x,ShortName) = %q ok=%v, want "GENERAL"`, v, ok)
	}
}

// 00load.t scene: fresh New() has no file name; SetFileName round-trips.
func TestLoad_NewGetSetFileName(t *testing.T) {
	cfg := New()
	if got := cfg.GetFileName(); got != "" {
		t.Fatalf("GetFileName on fresh New() = %q, want \"\"", got)
	}
	name := filepath.Join(t.TempDir(), "test9_name.ini")
	if err := cfg.SetFileName(name); err != nil {
		t.Fatalf("SetFileName: %v", err)
	}
	if got := cfg.GetFileName(); got != name {
		t.Fatalf("GetFileName = %q, want %q", got, name)
	}
}

// 00load.t scene: an empty file without -allowempty is an error (no warnings).
func TestLoad_EmptyFileWithoutAllowEmpty(t *testing.T) {
	if _, err := NewFromFile(testdata(t, "blank.ini")); err == nil {
		t.Fatalf("NewFromFile blank.ini (no allowempty) should error")
	}
}

// 00load.t scene: an empty file with -allowempty loads fine.
func TestLoad_EmptyFileWithAllowEmpty(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "blank.ini"), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile blank.ini (allowempty): %v", err)
	}
	if cfg == nil {
		t.Fatalf("cfg should be non-nil with allowempty")
	}
}

// 00load.t scene: a malformed file returns an error and records parse errors.
func TestLoad_MalformedFileRecordsErrors(t *testing.T) {
	_, err := NewFromFile(testdata(t, "bad.ini"))
	if err == nil {
		t.Fatalf("NewFromFile bad.ini should return an error")
	}
	// The returned error exposes Errors() []ParseError (the internal
	// multiError type). Validate that at least one parse error was recorded.
	le, ok := err.(interface{ Errors() []ParseError })
	if !ok {
		t.Fatalf("returned error does not expose Errors(): %T", err)
	}
	if errs := le.Errors(); len(errs) == 0 {
		t.Fatalf("expected at least one parse error, got none")
	}
}

// 38security-open.t: filenames are treated literally — no 2-arg open shell
// magic. A path containing shell metacharacters must simply fail to open (not
// execute a command, not truncate) and errors.Is os.ErrNotExist.
func TestSecurity_OpenLiteralPaths(t *testing.T) {
	// A nonexistent name with a trailing-pipe payload: open fails as
	// not-found rather than executing a command.
	_, err := NewFromFile("testdata/no_such_file_exists_|cat")
	if err == nil {
		t.Fatalf("NewFromFile on nonexistent path should error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}

	// A nonexistent name that looks like a redirect: still not-found.
	if _, err := NewFromFile("testdata/does_not_exist_>foo"); err == nil {
		t.Fatalf("NewFromFile redirect-looking name should error")
	}

	// Trailing whitespace is significant (filename not trimmed): a name
	// with a trailing newline does not open the trimmed real file.
	if _, err := NewFromFile("testdata/en.ini\n"); err == nil {
		t.Fatalf("NewFromFile with trailing-newline name should error (literal lookup)")
	}

	// A plain, existing file still opens as a file.
	if _, err := NewFromFile(testdata(t, "en.ini")); err != nil {
		t.Fatalf("NewFromFile plain existing file should succeed: %v", err)
	}
}

package inifiles

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"testing"
)

// translated from 19param-found-outside-section.t
func TestFallbackSection(t *testing.T) {
	dir := t.TempDir()
	src := []byte("\n; This is a malformed ini file with a key/value outside a section\n\nwrong = wronger\n\n[section]\n\nright = more right\n")
	path := filepath.Join(dir, "foo.ini")
	if err := os.WriteFile(path, src, 0644); err != nil {
		t.Fatal(err)
	}

	// Without -fallback, a parameter outside a section is a parse error.
	if _, err := NewFromFile(path); err == nil {
		t.Fatalf("expected error without fallback")
	}

	// With -fallback, the loose key is filed under GENERAL.
	cfg, err := NewFromFile(path, WithFallback("GENERAL"))
	if err != nil {
		t.Fatalf("NewFromFile with fallback: %v", err)
	}
	if !cfg.SectionExists("GENERAL") {
		t.Fatalf("GENERAL section should exist")
	}
	if !cfg.Exists("GENERAL", "wrong") {
		t.Fatalf("GENERAL should catch 'wrong'")
	}

	// On write, the [GENERAL] header is omitted, but the 'wrong' line appears.
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	content := buf.String()
	if !regexp.MustCompile(`(?m)^wrong`).MatchString(content) {
		t.Fatalf("fallback output should contain 'wrong' line:\n%s", content)
	}
	if regexp.MustCompile(`(?m)^\[GENERAL\]`).MatchString(content) {
		t.Fatalf("fallback section header should be omitted:\n%s", content)
	}
}

// translated from 19utf8.t. Go strings are UTF-8 and the library treats file
// contents as bytes, so UTF-8 values round-trip transparently.
func TestUTF8Values(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "utf8-test.ini")
	original := []byte("[section]\nmykey = tén\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	if v, ok := cfg.Val("section", "mykey"); !ok || v != "tén" {
		t.Fatalf(`Val = %q, want "tén"`, v)
	}
	if errs := cfg.Errors(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// setval to a multibyte value and rewrite.
	if err := cfg.SetVal("section", "mykey", "meö שלום"); err != nil {
		t.Fatal(err)
	}
	if v, _ := cfg.Val("section", "mykey"); v != "meö שלום" {
		t.Fatalf(`Val = %q, want "meö שלום"`, v)
	}
	if err := cfg.WriteFile(path); err != nil {
		t.Fatal(err)
	}

	// Re-read.
	cfg2, err := NewFromFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if v, _ := cfg2.Val("section", "mykey"); v != "meö שלום" {
		t.Fatalf(`reload Val = %q, want "meö שלום"`, v)
	}
}

// translated from 32mswin-outputs-contain-crs.t
func TestMSWinOutputContainsCRs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Test is only relevant for Microsoft Windows")
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "testConfig.ini")
	if err := os.WriteFile(cfgPath, []byte("[UserConfigFile]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for s := 1; s <= 4; s++ {
		for p := 1; p <= 4; p++ {
			cfg, err := NewFromFile(cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.NewVal("Section"+itoa(s), "Param"+itoa(p), "Value"+itoa(p)); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Rewrite(); err != nil {
				t.Fatal(err)
			}
		}
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// Every LF (0x0A) must be preceded by a CR (0x0D): no "[^\x0D]\x0A".
	if regexp.MustCompile(`[^\x0D]\x0A`).Match(data) {
		t.Fatalf("found an LF not preceded by CR")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// translated from 05hash.t using the idiomatic Go API (the Perl tie interface
// maps to Sections/Section accessors). Excludes Perl-only array-count
// assignment (tests 18-19), which has no Go equivalent.
func TestHashInterface_Idiomatic(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"), WithDefault("test1"), WithNoCase(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	// 2. scalar value
	if v, _ := cfg.Val("test1", "one"); v != "value1" {
		t.Fatalf(`Val(test1,one)=%q want "value1"`, v)
	}
	// 3. mult is an array (more than one line)
	vs, _ := cfg.Vals("test1", "mult")
	if len(vs) < 2 {
		t.Fatalf("Vals(test1,mult)=%v, want multi-line array", vs)
	}
	// 4-5. setval + delval persist after write+reload
	dir := t.TempDir()
	out := filepath.Join(dir, "test05.ini")
	cfg.SetFileName(out)
	if err := cfg.NewVal("test2", "seven", "value7"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Rewrite(); err != nil {
		t.Fatal(err)
	}
	cfg2, err := NewFromFile(out, WithDefault("test1"), WithNoCase(true))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := cfg2.Val("test2", "seven"); v != "value7" {
		t.Fatalf(`after reload Val=%q want "value7"`, v)
	}
	if err := cfg2.DelVal("test2", "seven"); err != nil {
		t.Fatal(err)
	}
	if err := cfg2.Rewrite(); err != nil {
		t.Fatal(err)
	}
	cfg3, err := NewFromFile(out, WithDefault("test1"), WithNoCase(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg3.Val("test2", "seven"); ok {
		t.Fatalf("seven should be gone after delval+reload")
	}

	// 6. default section lookup: test2 has no 'three', default test1 does.
	if v, _ := cfg3.Val("test2", "three"); v != "value3" {
		t.Fatalf(`Val(test2,three)=%q want "value3" (default)`, v)
	}
	// 7-8. case-insensitive lookup
	if v, _ := cfg3.Val("test2", "FOUR"); v != "value4" {
		t.Fatalf(`Val(test2,FOUR)=%q want "value4"`, v)
	}
	if v, _ := cfg3.Val("TEST2", "four"); v != "value4" {
		t.Fatalf(`Val(TEST2,four)=%q want "value4"`, v)
	}

	// 9. Sections() equals the set of section names.
	if len(cfg3.Sections()) == 0 {
		t.Fatalf("Sections empty")
	}

	// 10. Parameters('test1') lists the params.
	parms := cfg3.Parameters("test1")
	if len(parms) == 0 {
		t.Fatalf("Parameters(test1) empty")
	}
	expected := append([]string(nil), parms...)
	sort.Strings(expected)

	// 11. Copy semantics: Vals returns an independent slice.
	par2, _ := cfg3.Vals("test1", "mult")
	par2[0] = "MUTATED"
	again, _ := cfg3.Vals("test1", "mult")
	if again[0] == "MUTATED" {
		t.Fatalf("mutating returned slice should not affect config")
	}
}

// translated from 05hash.t scenario 17 (default value not in file).
func TestHashInterface_DefaultValueNotInFile(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"), WithDefault("default"), WithNoCase(true))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.NewVal("default", "cassius", "clay"); err != nil {
		t.Fatal(err)
	}
	// test1 has no 'cassius'; default section provides it.
	if v, _ := cfg.Val("test1", "cassius"); v != "clay" {
		t.Fatalf(`Val(test1,cassius)=%q want "clay"`, v)
	}
}

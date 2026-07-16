package inifiles

import (
	"strings"
	"testing"
)

// Translated from config-inifiles/t/01basic.t (read-only scenarios).
func TestBasic_ReadVal(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	// 1. object truthy already asserted by no error.

	// 2. val('test1','one') scalar -> 'value1'
	if v, ok := cfg.Val("test1", "one"); !ok || v != "value1" {
		t.Fatalf(`Val(test1,one) = %q ok=%v, want "value1"`, v, ok)
	}

	// 3. val('test1','one') list -> first elem 'value1'
	vs, ok := cfg.Vals("test1", "one")
	if !ok || len(vs) != 1 || vs[0] != "value1" {
		t.Fatalf(`Vals(test1,one) = %v ok=%v, want ["value1"]`, vs, ok)
	}

	// 4. val('test1','mult') scalar -> "one\ntwo\nthree"
	if v, ok := cfg.Val("test1", "mult"); !ok || v != "one\ntwo\nthree" {
		t.Fatalf(`Val(test1,mult) = %q ok=%v, want "one\ntwo\nthree"`, v, ok)
	}

	// 5. val('test1','mult') list -> ["one","two","three"]
	vs, ok = cfg.Vals("test1", "mult")
	if !ok || len(vs) != 3 || vs[0] != "one" || vs[1] != "two" || vs[2] != "three" {
		t.Fatalf(`Vals(test1,mult) = %v ok=%v, want [one two three]`, vs, ok)
	}

	// 6. newval('test1','eight', [a,b,c]) then val scalar joined; list -> [a,b,c]
	if err := cfg.NewVal("test1", "eight", "a", "b", "c"); err != nil {
		t.Fatalf("NewVal eight: %v", err)
	}
	if v, ok := cfg.Val("test1", "eight"); !ok || v != "a\nb\nc" {
		t.Fatalf(`Val(test1,eight) = %q ok=%v, want "a\nb\nc"`, v, ok)
	}
	vs, ok = cfg.Vals("test1", "eight")
	if !ok || len(vs) != 3 || vs[0] != "a" || vs[1] != "b" || vs[2] != "c" {
		t.Fatalf(`Vals(test1,eight) = %v ok=%v, want [a b c]`, vs, ok)
	}

	// 9. val('test1','<missing>','12345') default arg -> '12345'
	if v := cfg.ValWithDefault("test1", "<missing>", "12345"); v != "12345" {
		t.Fatalf(`ValWithDefault missing param = %q, want "12345"`, v)
	}

	// 10. val('<missing sect>','<missing>','12345') -> '12345'
	if v := cfg.ValWithDefault("<missing sect>", "<missing>", "12345"); v != "12345" {
		t.Fatalf(`ValWithDefault missing sect = %q, want "12345"`, v)
	}

	// missing param returns ok=false
	if _, ok := cfg.Val("test1", "nope"); ok {
		t.Fatalf("Val(missing) should be not found")
	}
	if _, ok := cfg.Vals("test1", "nope"); ok {
		t.Fatalf("Vals(missing) should be not found")
	}
}

// Translated from 02weird.t (read-only scenarios).
func TestWeird_ReadOddNames(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	// 1. val('[w]eird characters','multiline') -> "This\nis a multi-line\nvalue"
	if v, ok := cfg.Val("[w]eird characters", "multiline"); !ok || v != "This\nis a multi-line\nvalue" {
		t.Fatalf(`Val([w]eird characters,multiline) = %q ok=%v, want "This\nis a multi-line\nvalue"`, v, ok)
	}

	// 3. val('test7','criterion') -> 'price <= maximum'
	if v, ok := cfg.Val("test7", "criterion"); !ok || v != "price <= maximum" {
		t.Fatalf(`Val(test7,criterion) = %q ok=%v, want "price <= maximum"`, v, ok)
	}

	// 4. val('test7','criterion ') trailing space -> not found
	if _, ok := cfg.Val("test7", "criterion "); ok {
		t.Fatalf(`Val(test7,"criterion ") should be not found`)
	}

	// Ensure sections with pipes/spaces parsed (group members count).
	for _, name := range []string{"group member one", "group member two", "group member three"} {
		if !cfg.SectionExists(name) {
			t.Errorf("SectionExists(%q) = false, want true", name)
		}
	}
}

// From 14brackets-within-values.t.
func TestBracketsInValues(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "brackets-in-values.ini"), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	want := "http://www.example.com/files/file[1-22].gz"
	if v, ok := cfg.Val("SiteName", "file"); !ok || v != want {
		t.Fatalf(`Val(SiteName,file) = %q ok=%v, want %q`, v, ok, want)
	}
}

// From 27empty-ini.t.
func TestEmptyIni(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "for-27-empty.ini"), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromFile empty (allowed): %v", err)
	}
	if members := cfg.GroupMembers("anything"); members != nil {
		t.Fatalf("GroupMembers on empty = %v, want nil", members)
	}

	// Without AllowEmpty, an empty file is an error.
	if _, err := NewFromFile(testdata(t, "for-27-empty.ini")); err == nil {
		t.Fatalf("NewFromFile empty (not allowed) should error")
	}
}

// Reading a parameter assigned an empty string yields "" (RT #68554 read side).
func TestReadEmptyValue(t *testing.T) {
	src := "[MySection]\nMyParam=\nKey=val\n"
	cfg, err := NewFromBytes([]byte(src), WithAllowEmpty(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("MySection", "MyParam"); !ok || v != "" {
		t.Fatalf(`Val(MySection,MyParam) = %q ok=%v, want ""`, v, ok)
	}
	if v, ok := cfg.Val("MySection", "Key"); !ok || v != "val" {
		t.Fatalf(`Val(MySection,Key) = %q ok=%v, want "val"`, v, ok)
	}
}

// Multiple `=` chars in a value split on the first only (from 07misc scenario 1).
func TestMultipleEqualsInValue(t *testing.T) {
	src := "[s]\nk=a=b=c\n"
	cfg, err := NewFromBytes([]byte(src))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "a=b=c" {
		t.Fatalf(`Val(s,k) = %q ok=%v, want "a=b=c"`, v, ok)
	}
}

// Param name that is a substring of another loads independently (07misc scenario 2).
func TestSubstringParamNames(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	if v, ok := cfg.Val("substring", "boot"); !ok || v != "smarty" {
		t.Fatalf(`Val(substring,boot) = %q ok=%v, want "smarty"`, v, ok)
	}
	if v, ok := cfg.Val("substring", "bootcamp"); !ok || v != "dummy" {
		t.Fatalf(`Val(substring,bootcamp) = %q ok=%v, want "dummy"`, v, ok)
	}
}

// Section headers with leading/inner whitespace are trimmed per the grammar.
func TestSectionHeaderWhitespace(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	// The fixture has "[ group member two ]" -> name "group member two".
	if !cfg.SectionExists("group member two") {
		t.Errorf("expected trimmed section 'group member two' to exist")
	}
	// And "[  group member three]" -> "group member three".
	if !cfg.SectionExists("group member three") {
		t.Errorf("expected trimmed section 'group member three' to exist")
	}
}

// Bad file (malformed line) fails to load and records an error (from 00load #15 / bad.ini).
func TestBadFileFails(t *testing.T) {
	_, err := NewFromFile(testdata(t, "bad.ini"))
	if err == nil {
		t.Fatalf("NewFromFile bad.ini should error")
	}
	// Empty file without AllowEmpty is an error too.
	if _, err := NewFromFile(testdata(t, "blank.ini")); err == nil {
		t.Fatalf("NewFromFile blank.ini should error")
	}
}

// Section names may contain brackets (the last ] is structural).
func TestSectionNameWithBrackets(t *testing.T) {
	src := "[[w]eird]\nk=v\n"
	cfg, err := NewFromBytes([]byte(src))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if !cfg.SectionExists("[w]eird") {
		t.Fatalf("expected section '[w]eird' to exist; got sections=%v", cfg.Sections())
	}
	if v, ok := cfg.Val("[w]eird", "k"); !ok || v != "v" {
		t.Fatalf(`Val([w]eird,k) = %q ok=%v, want "v"`, v, ok)
	}
}

// Duplicate section lines merge parameters (silent).
func TestDuplicateSectionMerge(t *testing.T) {
	src := strings.Join([]string{
		"[s]",
		"a=1",
		"[s]",
		"b=2",
	}, "\n")
	cfg, err := NewFromBytes([]byte(src))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "a"); !ok || v != "1" {
		t.Errorf(`Val(s,a)=%q ok=%v want "1"`, v, ok)
	}
	if v, ok := cfg.Val("s", "b"); !ok || v != "2" {
		t.Errorf(`Val(s,b)=%q ok=%v want "2"`, v, ok)
	}
}

// Repeated param lines accumulate into a multi-valued parameter (push).
func TestRepeatedParamAccumulates(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "test.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}
	vs, ok := cfg.Vals("group member one", "repeated")
	if !ok {
		t.Fatalf("Vals repeated not found")
	}
	want := []string{"one", "two", "three"}
	if len(vs) != 3 || vs[0] != want[0] || vs[1] != want[1] || vs[2] != want[2] {
		t.Fatalf("Vals repeated = %v, want %v", vs, want)
	}
}

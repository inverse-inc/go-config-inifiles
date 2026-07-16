package inifiles

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// SetSectionComment early-returns when sect is empty (ErrEmptyArg) or no
// comments are given (0,nil); DeleteSectionComment is a no-op (returns nil) on a
// missing section. Covers the remaining branches of the comments accessors.
func TestCoverageCommentsEarlyReturns(t *testing.T) {
	cfg := New()

	if n, err := cfg.SetSectionComment(""); n != 0 || !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetSectionComment(\"\") = (%d,%v), want (0,ErrEmptyArg)", n, err)
	}
	if cfg.SectionExists("") {
		t.Errorf("SetSectionComment(\"\") should not create a section")
	}
	if n, err := cfg.SetSectionComment("s"); n != 0 || err != nil {
		t.Errorf("SetSectionComment(\"s\") with no comments = (%d,%v), want (0,nil)", n, err)
	}
	if cfg.SectionExists("s") {
		t.Errorf("SetSectionComment with no comments should not create a section")
	}

	if err := cfg.DeleteSectionComment("nonexistent"); err != nil {
		t.Errorf("DeleteSectionComment on missing section = %v, want nil", err)
	}
}

// splitLines strips a leading UTF-8 BOM before splitting.
func TestCoverageSplitLinesBOM(t *testing.T) {
	data := []byte{0xEF, 0xBB, 0xBF, '[', 's', ']', '\n', 'k', '=', 'v', '\n'}
	cfg, err := NewFromBytes(data)
	if err != nil {
		t.Fatalf("NewFromBytes with BOM: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "v" {
		t.Errorf("Val(s,k) after BOM strip = (%q,%v), want (v,true)", v, ok)
	}
}

// isBlank covers its whitespace case (a line of only spaces/tabs, not the empty
// string, which skips the loop entirely).
func TestCoverageIsBlankWhitespace(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[s]\n \t \nk=v\n"))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "v" {
		t.Errorf("Val(s,k) = (%q,%v), want (v,true)", v, ok)
	}
}

// splitTrailingComment's fall-through (no allowed comment rune in the value)
// returns the value verbatim with an empty trailing comment. Reached only when
// HandleTrailingComment is enabled.
func TestCoverageSplitTrailingCommentNoComment(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[s]\nk=hello\n"), WithTrailingComment(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "hello" {
		t.Errorf("Val(s,k) = (%q,%v), want (hello,true)", v, ok)
	}
	if tc, ok := cfg.GetParameterTrailingComment("s", "k"); !ok || tc != "" {
		t.Errorf("trailing comment = (%q,%v), want (\"\",true)", tc, ok)
	}
}

// processContinue strips a trailing backslash and, when there is no further
// line (EOF), returns without erroring.
func TestCoverageProcessContinueEOF(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[s]\nk=foo\\\n"), WithAllowContinue(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "foo" {
		t.Errorf("Val(s,k) = (%q,%v), want (foo,true)", v, ok)
	}
}

// unwrapPHPQuotes returns the value unchanged when it does not start with a
// quote, and when the opening quote is not matched at the end.
func TestCoverageUnwrapPHPQuotes(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"no quote", "hello", "hello"},
		{"mismatched quotes", `"abc'`, `"abc'`},
		{"matched double unwrap", `"abc"`, `abc`},
		{"matched double collapsed", `"a""b"`, `ab`},
		{"matched single unwrap", `'abc'`, `abc`},
		{"matched single collapsed", `'a''b'`, `ab`},
		{"escaped double quote", `"a\"b"`, `a"b`},
	}
	for _, tc := range cases {
		if got := unwrapPHPQuotes(tc.input); got != tc.want {
			t.Errorf("unwrapPHPQuotes(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}

	// Also exercise via a parsed PHPCompat config to cover the call sites.
	cfg, err := NewFromBytes([]byte(`[s]
k=hello
k2="abc'
`), WithPHPCompat(true))
	if err != nil {
		t.Fatalf("NewFromBytes PHPCompat: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "hello" {
		t.Errorf("PHPCompat Val(s,k) = (%q,%v), want (hello,true)", v, ok)
	}
	if v, ok := cfg.Val("s", "k2"); !ok || v != `"abc'` {
		t.Errorf("PHPCompat Val(s,k2) = (%q,%v), want (%q,true)", v, ok, `"abc'`)
	}
}

// replaceAll returns its input unchanged when old is empty.
func TestCoverageReplaceAllEmptyOld(t *testing.T) {
	if got := replaceAll("abc", "", "X"); got != "abc" {
		t.Errorf("replaceAll(abc,\"\",X) = %q, want abc", got)
	}
	if got := replaceAll("aXbXc", "X", "Y"); got != "aYbYc" {
		t.Errorf("replaceAll = %q, want aYbYc", got)
	}
}

// newvalRaw stores an empty scalar value when a here-doc is closed immediately
// (no body lines), exercising the len(vals)==0 branch.
func TestCoverageNewvalRawEmptyVals(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[s]\nk= <<EOT\nEOT\n"))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "" {
		t.Errorf("Val(s,k) = (%q,%v), want (\"\",true)", v, ok)
	}
}

// loadValue is a no-op when the current section does not resolve.
func TestCoverageLoadValueNilSection(t *testing.T) {
	c := New()
	ps := &parserState{currSect: "missing"}
	c.loadValue(ps, []string{"v"}, "", false)
	if s := c.sectionOf("missing"); s != nil {
		t.Errorf("loadValue should not have created the missing section")
	}
}

// pushRaw promotes an empty (nil) lines slice to [""] before appending.
func TestCoveragePushRawEmptyLines(t *testing.T) {
	c := New()
	s := &section{name: "s"}
	p := &param{name: "k", val: value{lines: nil, isMulti: false}}
	c.pushRaw(s, p, []string{"x"})
	want := []string{"", "x"}
	if !sliceEq(p.val.lines, want) {
		t.Errorf("pushRaw empty-lines: p.val.lines = %v, want %v", p.val.lines, want)
	}
	if !p.val.isMulti {
		t.Errorf("pushRaw should mark the value multi")
	}
	if !p.hasEOT || p.eot != "EOT" {
		t.Errorf("pushRaw should set the default EOT marker, got hasEOT=%v eot=%q", p.hasEOT, p.eot)
	}
}

// PushVal strips the PHP "[]" tail from parameter names under PHPCompat.
func TestCoveragePushValPHPCompat(t *testing.T) {
	c := New(WithPHPCompat(true))
	if err := c.NewVal("s", "arr[]", "a"); err != nil {
		t.Fatal(err)
	}
	if err := c.PushVal("s", "arr[]", "b"); err != nil {
		t.Fatal(err)
	}
	vs, _ := c.Vals("s", "arr")
	if !sliceEq(vs, []string{"a", "b"}) {
		t.Errorf("PushVal PHPCompat Vals = %v, want [a b]", vs)
	}
}

// PushVal promotes an empty lines slice to [""] before appending, the way
// pushRaw does. We set up the empty-lines param manually since the public API
// always yields a non-empty scalar.
func TestCoveragePushValEmptyLines(t *testing.T) {
	c := New()
	c.AddSection("s")
	s := c.sectionOf("s")
	s.addParam(&param{name: "k", val: value{lines: nil}})
	if err := c.PushVal("s", "k", "x"); err != nil {
		t.Fatalf("PushVal: %v", err)
	}
	vs, _ := c.Vals("s", "k")
	if !sliceEq(vs, []string{"", "x"}) {
		t.Errorf("PushVal empty-lines Vals = %v, want [\"\" x]", vs)
	}
}

// applyImport copies the imported config's groups and end-of-file comments into
// the importing config. New() triggers applyImport eagerly.
func TestCoverageApplyImportGroupsAndEndComments(t *testing.T) {
	imp, err := NewFromBytes([]byte("[g one]\nk=v\n# trailing end comment\n"))
	if err != nil {
		t.Fatalf("NewFromBytes imported: %v", err)
	}
	if groups := imp.Groups(); len(groups) != 1 || groups[0] != "g" {
		t.Fatalf("imported groups = %v, want [g]", groups)
	}

	c := New(WithImport(imp))
	if groups := c.Groups(); len(groups) != 1 || groups[0] != "g" {
		t.Errorf("imported groups after applyImport = %v, want [g]", groups)
	}
	if members := c.GroupMembers("g"); len(members) != 1 || members[0] != "g one" {
		t.Errorf("GroupMembers(g) = %v, want [g one]", members)
	}
	if len(c.endComments) != 1 || c.endComments[0] != "# trailing end comment" {
		t.Errorf("endComments = %v, want [# trailing end comment]", c.endComments)
	}
}

// writeTo's non-delta path skips deleted sections, deleted params, and params
// with a nil value.
func TestCoverageWriteToDeletedGuardsAndIsNil(t *testing.T) {
	c := New()
	live := &section{name: "live"}
	live.params = append(live.params,
		&param{name: "lp", val: singleValue("lv")},
		&param{name: "pdel", deleted: true, val: singleValue("x")},
		&param{name: "nparam", val: value{lines: nil}},
	)
	c.sections = append(c.sections, live)
	c.sectIdx["live"] = live

	dels := &section{name: "delsect", deleted: true}
	c.sections = append(c.sections, dels)
	c.sectIdx["delsect"] = dels

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[live]") || !strings.Contains(out, "lp=lv") {
		t.Errorf("output missing live section/param;\n%s", out)
	}
	for _, bad := range []string{"pdel", "nparam", "delsect"} {
		if strings.Contains(out, bad) {
			t.Errorf("output should omit %q;\n%s", bad, out)
		}
	}
}

// writeParam emits "name=" (no value) when outputLines yields zero lines, which
// requires a multi-valued param whose lines slice is non-nil but empty.
func TestCoverageWriteParamEmptyVals(t *testing.T) {
	c := New()
	s := &section{name: "s"}
	s.params = append(s.params, &param{name: "k", val: value{lines: []string{}, isMulti: true}})
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "k=\n") {
		t.Errorf("output missing k= line;\n%s", out)
	}
}

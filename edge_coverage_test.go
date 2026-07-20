package inifiles

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// --- parse.go helpers direct unit tests ---

func TestTrimTrailingWhitespace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"hello  ", "hello"},
		{"hello\t\t", "hello"},
		{"hello \t ", "hello"},
		{"  ", ""},
		{"", ""},
		{"a\tb  ", "a\tb"},
	}
	for _, tc := range cases {
		if got := trimTrailingWhitespace(tc.in); got != tc.want {
			t.Errorf("trimTrailingWhitespace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTrimLeadingWhitespace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"  hello", "hello"},
		{"\t\thello", "hello"},
		{" \t hello", "hello"},
		{"  ", ""},
		{"", ""},
		{"a b  ", "a b  "},
	}
	for _, tc := range cases {
		if got := trimLeadingWhitespace(tc.in); got != tc.want {
			t.Errorf("trimLeadingWhitespace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHereDocMarker(t *testing.T) {
	cases := []struct {
		in     string
		wantS  string
		wantOk bool
	}{
		{"<<EOT", "EOT", true},
		{"<<MARK", "MARK", true},
		{"<<", "", true},
		{"<EOT", "", false},
		{"", "", false},
		{"<", "", false},
		{"hello", "", false},
	}
	for _, tc := range cases {
		s, ok := hereDocMarker(tc.in)
		if ok != tc.wantOk || s != tc.wantS {
			t.Errorf("hereDocMarker(%q) = (%q, %v), want (%q, %v)", tc.in, s, ok, tc.wantS, tc.wantOk)
		}
	}
}

func TestIsBlankCR(t *testing.T) {
	if !isBlank("\r") {
		t.Errorf("isBlank(\\r) = false, want true")
	}
	if !isBlank("\r\r\r") {
		t.Errorf("isBlank(\\r\\r\\r) = false, want true")
	}
	if !isBlank(" \t\r") {
		t.Errorf("isBlank(\" \\t\\r\") = false, want true")
	}
	if isBlank("a") {
		t.Errorf("isBlank(a) = true, want false")
	}
}

func TestSplitTrailingCommentMultipleDelimiters(t *testing.T) {
	allowed := map[rune]struct{}{'#': {}, ';': {}}
	val, ec := splitTrailingComment("hello ; world", allowed)
	if val != "hello" || ec != "world" {
		t.Errorf("splitTrailingComment = (%q, %q), want (hello, world)", val, ec)
	}
	allowed2 := map[rune]struct{}{'#': {}}
	val2, ec2 := splitTrailingComment("hello ; world", allowed2)
	if val2 != "hello ; world" || ec2 != "" {
		t.Errorf("splitTrailingComment (# only) = (%q, %q), want (hello ; world, \"\")", val2, ec2)
	}
}

func TestProcessContinueMultipleLines(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[s]\nk=line1\\\nline2\\\nline3\n"),
		WithAllowContinue(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}
	v, ok := cfg.Val("s", "k")
	if !ok {
		t.Fatal("Val not found")
	}
	if v != "line1line2line3" {
		t.Errorf("Val = %q, want line1line2line3", v)
	}
}

// --- write.go helpers direct unit tests ---

func TestCalcEOTMarkMultiCollision(t *testing.T) {
	cfg := New()
	p := &param{name: "x", hasEOT: true, eot: "M"}
	// Value contains "M", "MA", "MAA" — marker should grow past all.
	got := cfg.calcEOTMark(p, []string{"has M MA MAA"})
	if got != "MAAA" {
		t.Errorf("calcEOTMark multi-collision = %q, want MAAA", got)
	}
}

func TestOutputLinesWithLineEndings(t *testing.T) {
	cfg := New()
	cfg.lineEnds = []byte{'\r', '\n'}

	s := &section{name: "s"}
	p := &param{name: "k", val: singleValue("a\rb\nc")}
	s.params = append(s.params, p)
	cfg.sections = append(cfg.sections, s)
	cfg.sectIdx["s"] = s

	got := cfg.outputLines(p)
	// \r and \n are both in the orRunes set, so the value is split on both.
	want := []string{"a", "b", "c"}
	if !sliceEq(got, want) {
		t.Errorf("outputLines with \\r\\n endings = %v, want %v", got, want)
	}
}

func TestOutputLinesMultiValued(t *testing.T) {
	cfg := New()
	p := &param{name: "k", val: multiValue([]string{"x", "y"})}
	got := cfg.outputLines(p)
	if !sliceEq(got, []string{"x", "y"}) {
		t.Errorf("outputLines multi = %v, want [x y]", got)
	}
}

func TestOrRunesTripleEnding(t *testing.T) {
	cfg := New()
	cfg.lineEnds = []byte{'\r', '\n', 0x15}
	got := cfg.orRunes()
	if len(got) != 3 {
		t.Errorf("orRunes triple = %v, want 3 runes", got)
	}
}

func TestSplitOnOrsRunesConsecutiveDelimiters(t *testing.T) {
	got := splitOnOrsRunes("a\n\nb", []rune{'\n'})
	want := []string{"a", "", "b"}
	if !sliceEq(got, want) {
		t.Errorf("splitOnOrsRunes consecutive = %v, want %v", got, want)
	}
}

func TestSplitOnOrsRunesNoMatch(t *testing.T) {
	got := splitOnOrsRunes("hello", []rune{'\n'})
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("splitOnOrsRunes no match = %v, want [hello]", got)
	}
}

// --- config.go edge cases ---

func TestNewFromFileNotFound(t *testing.T) {
	_, err := NewFromFile("/no/such/file/exists.ini")
	if err == nil {
		t.Errorf("NewFromFile nonexistent = nil, want error")
	}
}

func TestNewFromBytesEmpty(t *testing.T) {
	_, err := NewFromBytes([]byte{})
	if err == nil {
		t.Errorf("NewFromBytes empty = nil, want error")
	}
}

// --- sections.go edge cases ---

func TestSectionExistsEmpty(t *testing.T) {
	cfg := New()
	if cfg.SectionExists("") {
		t.Errorf("SectionExists(\"\") = true, want false")
	}
}

func TestSectionOfEmpty(t *testing.T) {
	cfg := New()
	if s := cfg.sectionOf(""); s != nil {
		t.Errorf("sectionOf(\"\") = %v, want nil", s)
	}
}

func TestAddSectionEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.AddSection(""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("AddSection(\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestDeleteSectionEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteSection(""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("DeleteSection(\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestDeleteSectionMissing(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteSection("ghost"); err != nil {
		t.Errorf("DeleteSection(ghost) = %v, want nil", err)
	}
}

func TestCopySectionEmptyArgs(t *testing.T) {
	cfg := New()
	if err := cfg.CopySection("", "new", false); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("CopySection(\"\",new) = %v, want ErrEmptyArg", err)
	}
	if err := cfg.CopySection("old", "", false); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("CopySection(old,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestCopySectionMissing(t *testing.T) {
	cfg := New()
	if err := cfg.CopySection("ghost", "new", false); !errors.Is(err, ErrNotFound) {
		t.Errorf("CopySection(ghost,new) = %v, want ErrNotFound", err)
	}
}

func TestCopySectionAlreadyExists(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	if err := cfg.CopySection("s", "s", false); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("CopySection(s,s) = %v, want ErrAlreadyExists", err)
	}
}

func TestRenameSectionEmptyArgsEdge(t *testing.T) {
	cfg := New()
	if err := cfg.RenameSection("", "new", false); err == nil {
		t.Errorf("RenameSection(\"\") = nil, want error")
	}
	if err := cfg.RenameSection("old", "", false); err == nil {
		t.Errorf("RenameSection(_, \"\") = nil, want error")
	}
}

func TestParametersMissing(t *testing.T) {
	cfg := New()
	if ps := cfg.Parameters("ghost"); ps != nil {
		t.Errorf("Parameters(ghost) = %v, want nil", ps)
	}
}

func TestParametersEmpty(t *testing.T) {
	cfg := New()
	if ps := cfg.Parameters(""); ps != nil {
		t.Errorf("Parameters(\"\") = %v, want nil", ps)
	}
}

func TestIsImportedEmptyArgs(t *testing.T) {
	cfg := New()
	if cfg.IsImported("", "p") {
		t.Errorf("IsImported(\"\",p) = true, want false")
	}
	if cfg.IsImported("s", "") {
		t.Errorf("IsImported(s,\"\") = true, want false")
	}
}

func TestIsImportedMissingSection(t *testing.T) {
	cfg := New()
	if cfg.IsImported("ghost", "p") {
		t.Errorf("IsImported(ghost,p) = true, want false")
	}
}

func TestIsImportedNonImportedSection(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "p", "v")
	if cfg.IsImported("s", "p") {
		t.Errorf("IsImported on non-imported section = true, want false")
	}
}

func TestMyParametersMissing(t *testing.T) {
	cfg := New()
	if ps := cfg.MyParameters("ghost"); ps != nil {
		t.Errorf("MyParameters(ghost) = %v, want nil", ps)
	}
}

func TestMyParametersNonImported(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "p1", "v1")
	cfg.NewVal("s", "p2", "v2")
	ps := cfg.MyParameters("s")
	if !sliceEq(ps, []string{"p1", "p2"}) {
		t.Errorf("MyParameters(s) = %v, want [p1 p2]", ps)
	}
}

func TestResortSectionsEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.ResortSections(nil); err != nil {
		t.Errorf("ResortSections(nil) = %v, want nil", err)
	}
	if err := cfg.ResortSections([]string{"only"}); err != nil {
		t.Errorf("ResortSections([only]) = %v, want nil", err)
	}
}

func TestResortSectionsMissing(t *testing.T) {
	cfg := New()
	cfg.AddSection("a")
	cfg.AddSection("b")
	if err := cfg.ResortSections([]string{"a", "ghost"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("ResortSections with ghost = %v, want ErrNotFound", err)
	}
}

func TestResortSectionsPartial(t *testing.T) {
	cfg := New()
	cfg.AddSection("a")
	cfg.AddSection("b")
	cfg.AddSection("c")
	if err := cfg.ResortSections([]string{"c", "a"}); err != nil {
		t.Fatalf("ResortSections: %v", err)
	}
	got := cfg.Sections()
	if !sliceEq(got, []string{"b", "c", "a"}) {
		t.Errorf("partial resort = %v, want [b c a]", got)
	}
}

// --- param.go edge cases ---

func TestValEmptyArgs(t *testing.T) {
	cfg := New()
	if _, ok := cfg.Val("", "p"); ok {
		t.Errorf("Val(\"\") found, want not found")
	}
	if _, ok := cfg.Val("s", ""); ok {
		t.Errorf("Val(_,\"\") found, want not found")
	}
}

func TestValsEmptyArgs(t *testing.T) {
	cfg := New()
	if vs, ok := cfg.Vals("", "p"); ok {
		t.Errorf("Vals(\"\") = %v, want not found", vs)
	}
	if vs, ok := cfg.Vals("s", ""); ok {
		t.Errorf("Vals(_,\"\") = %v, want not found", vs)
	}
}

func TestExistsEmptyArgs(t *testing.T) {
	cfg := New()
	if cfg.Exists("", "p") {
		t.Errorf("Exists(\"\") = true, want false")
	}
	if cfg.Exists("s", "") {
		t.Errorf("Exists(_,\"\") = true, want false")
	}
}

func TestExistsMissing(t *testing.T) {
	cfg := New()
	if cfg.Exists("ghost", "p") {
		t.Errorf("Exists(ghost,p) = true, want false")
	}
}

func TestNewValEmptyArgs(t *testing.T) {
	cfg := New()
	if err := cfg.NewVal("", "p", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("NewVal(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.NewVal("s", "", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("NewVal(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestSetValEmptyArgs(t *testing.T) {
	cfg := New()
	if err := cfg.SetVal("", "p", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetVal(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.SetVal("s", "", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetVal(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestSetValMissingSection(t *testing.T) {
	cfg := New()
	if err := cfg.SetVal("ghost", "p", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetVal(ghost,p) = %v, want ErrNotFound", err)
	}
}

func TestSetValMissingParam(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	if err := cfg.SetVal("s", "ghost", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetVal(s,ghost) = %v, want ErrNotFound", err)
	}
}

func TestPushValEmptyArgs(t *testing.T) {
	cfg := New()
	if err := cfg.PushVal("", "p", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("PushVal(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.PushVal("s", "", "v"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("PushVal(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestPushValMissingSection(t *testing.T) {
	cfg := New()
	if err := cfg.PushVal("ghost", "p", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("PushVal(ghost,p) = %v, want ErrNotFound", err)
	}
}

func TestPushValMissingParam(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	if err := cfg.PushVal("s", "ghost", "v"); !errors.Is(err, ErrNotFound) {
		t.Errorf("PushVal(s,ghost) = %v, want ErrNotFound", err)
	}
}

func TestPushValNoValues(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "p", "v")
	if err := cfg.PushVal("s", "p"); err != nil {
		t.Errorf("PushVal no values = %v, want nil", err)
	}
}

func TestDelValEmptyArgs(t *testing.T) {
	cfg := New()
	if err := cfg.DelVal("", "p"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("DelVal(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.DelVal("s", ""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("DelVal(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestDelValMissingSection(t *testing.T) {
	cfg := New()
	if err := cfg.DelVal("ghost", "p"); err != nil {
		t.Errorf("DelVal(ghost,p) = %v, want nil", err)
	}
}

func TestDelValMissingParam(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	if err := cfg.DelVal("s", "ghost"); err != nil {
		t.Errorf("DelVal(s,ghost) = %v, want nil", err)
	}
}

func TestFindParamMissing(t *testing.T) {
	cfg := New()
	if p := cfg.findParam("ghost", "p"); p != nil {
		t.Errorf("findParam(ghost,p) = %v, want nil", p)
	}
}

func TestStripPHPTailArray(t *testing.T) {
	cases := []struct{ in, want string }{
		{"arr[]", "arr"},
		{"arr", "arr"},
		{"[]", ""},
		{"", ""},
		{"a[b]", "a[b]"},
	}
	for _, tc := range cases {
		if got := stripPHPTailArray(tc.in); got != tc.want {
			t.Errorf("stripPHPTailArray(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- comments.go edge cases ---

func TestSetSectionCommentEmpty(t *testing.T) {
	cfg := New()
	if _, err := cfg.SetSectionComment(""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetSectionComment(\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestSetSectionCommentNoComments(t *testing.T) {
	cfg := New()
	if n, err := cfg.SetSectionComment("s"); n != 0 || err != nil {
		t.Errorf("SetSectionComment(\"s\") = (%d,%v), want (0,nil)", n, err)
	}
}

func TestGetSectionCommentMissing(t *testing.T) {
	cfg := New()
	if cs, ok := cfg.GetSectionComment("ghost"); ok || len(cs) != 0 {
		t.Errorf("GetSectionComment(ghost) = (%v, %v), want nil/empty", cs, ok)
	}
}

func TestDeleteSectionCommentMissing(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteSectionComment("ghost"); err != nil {
		t.Errorf("DeleteSectionComment(ghost) = %v, want nil", err)
	}
}

func TestSetParameterCommentEmpty(t *testing.T) {
	cfg := New()
	if _, err := cfg.SetParameterComment("", "p"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterComment(\"\") = %v, want ErrEmptyArg", err)
	}
	if _, err := cfg.SetParameterComment("s", ""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterComment(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestGetParameterCommentMissing(t *testing.T) {
	cfg := New()
	if cs, ok := cfg.GetParameterComment("ghost", "p"); ok || len(cs) != 0 {
		t.Errorf("GetParameterComment(ghost,p) = (%v, %v), want nil/empty", cs, ok)
	}
}

func TestDeleteParameterCommentMissing(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteParameterComment("ghost", "p"); err != nil {
		t.Errorf("DeleteParameterComment(ghost,p) = %v, want nil", err)
	}
}

func TestSetParameterTrailingCommentEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.SetParameterTrailingComment("", "p", "cmt"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterTrailingComment(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.SetParameterTrailingComment("s", "", "cmt"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterTrailingComment(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestSetParameterTrailingCommentMissing(t *testing.T) {
	cfg := New()
	if err := cfg.SetParameterTrailingComment("ghost", "p", "cmt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetParameterTrailingComment(ghost,p) = %v, want ErrNotFound", err)
	}
}

func TestGetParameterTrailingCommentEmpty(t *testing.T) {
	cfg := New()
	if _, ok := cfg.GetParameterTrailingComment("", "p"); ok {
		t.Errorf("GetParameterTrailingComment(\"\") found, want not found")
	}
	if _, ok := cfg.GetParameterTrailingComment("s", ""); ok {
		t.Errorf("GetParameterTrailingComment(_,\"\") found, want not found")
	}
}

func TestGetParameterTrailingCommentMissing(t *testing.T) {
	cfg := New()
	if _, ok := cfg.GetParameterTrailingComment("ghost", "p"); ok {
		t.Errorf("GetParameterTrailingComment(ghost,p) found, want not found")
	}
}

// --- eot.go edge cases ---

func TestSetParameterEOTEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.SetParameterEOT("", "p", "EOT"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterEOT(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.SetParameterEOT("s", "", "EOT"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterEOT(_,\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.SetParameterEOT("s", "p", ""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetParameterEOT(_,_,\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestGetParameterEOTMissing(t *testing.T) {
	cfg := New()
	if got := cfg.GetParameterEOT("ghost", "p"); got != "" {
		t.Errorf("GetParameterEOT(ghost,p) = %q, want empty", got)
	}
}

func TestDeleteParameterEOTEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.DeleteParameterEOT("", "p"); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("DeleteParameterEOT(\"\") = %v, want ErrEmptyArg", err)
	}
	if err := cfg.DeleteParameterEOT("s", ""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("DeleteParameterEOT(_,\"\") = %v, want ErrEmptyArg", err)
	}
}

// --- groups.go edge cases ---

func TestGroupsEmpty(t *testing.T) {
	cfg := New()
	if gs := cfg.Groups(); len(gs) != 0 {
		t.Errorf("Groups() = %v, want []", gs)
	}
}

func TestGroupMembersMissing(t *testing.T) {
	cfg := New()
	if ms := cfg.GroupMembers("ghost"); len(ms) != 0 {
		t.Errorf("GroupMembers(ghost) = %v, want []", ms)
	}
}

func TestSetGroupMemberEmpty(t *testing.T) {
	cfg := New()
	// SetGroupMember takes a section name; empty string is a no-op.
	cfg.SetGroupMember("")
}

func TestRemoveGroupMemberEmpty(t *testing.T) {
	cfg := New()
	// RemoveGroupMember takes a section name; empty string is a no-op.
	cfg.RemoveGroupMember("")
}

// --- write.go edge cases ---

func TestWriteToDeltaWithDeletions(t *testing.T) {
	c := New()
	c.NewVal("s1", "k1", "v1")
	c.DelVal("s1", "k1")
	c.NewVal("s2", "k2", "v2")

	var buf bytes.Buffer
	if _, err := c.WriteToDelta(&buf); err != nil {
		t.Fatalf("WriteToDelta: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "k1 is deleted") {
		t.Errorf("delta missing deletion marker;\n%s", out)
	}
	if !strings.Contains(out, "k2=v2") {
		t.Errorf("delta missing live param;\n%s", out)
	}
}

func TestWriteParamNoMultiline(t *testing.T) {
	c := New(WithNoMultiline(true))
	s := &section{name: "s"}
	p := &param{name: "k", val: multiValue([]string{"a", "b", "c"})}
	s.params = append(s.params, p)
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	// NoMultiline: should output repeated "k=a\nk=b\nk=c\n" instead of here-doc.
	if strings.Contains(out, "<<") {
		t.Errorf("NoMultiline output should not contain <<;\n%s", out)
	}
	if !strings.Contains(out, "k=a") || !strings.Contains(out, "k=b") || !strings.Contains(out, "k=c") {
		t.Errorf("NoMultiline output missing values;\n%s", out)
	}
}

func TestWriteParamSingleValued(t *testing.T) {
	c := New()
	s := &section{name: "s"}
	p := &param{name: "k", val: singleValue("hello")}
	s.params = append(s.params, p)
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "k=hello") {
		t.Errorf("output missing k=hello;\n%s", out)
	}
}

func TestWriteToWithSectionComments(t *testing.T) {
	c := New()
	s := &section{name: "s", comments: []string{"# section comment"}}
	p := &param{name: "k", val: singleValue("v")}
	s.params = append(s.params, p)
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# section comment") {
		t.Errorf("output missing section comment;\n%s", out)
	}
}

func TestWriteToWithParamComments(t *testing.T) {
	c := New()
	s := &section{name: "s"}
	p := &param{name: "k", val: singleValue("v"), comments: []string{"# param comment"}}
	s.params = append(s.params, p)
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# param comment") {
		t.Errorf("output missing param comment;\n%s", out)
	}
}

func TestWriteToEndComments(t *testing.T) {
	c := New()
	c.endComments = []string{"# end of file comment"}
	s := &section{name: "s"}
	s.params = append(s.params, &param{name: "k", val: singleValue("v")})
	c.sections = append(c.sections, s)
	c.sectIdx["s"] = s

	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# end of file comment") {
		t.Errorf("output missing end comment;\n%s", out)
	}
}

func TestWriteFileEmpty(t *testing.T) {
	cfg := New()
	dir := t.TempDir()
	path := dir + "/empty.ini"
	if err := cfg.WriteFile(path); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}
	data, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(data) != "" {
		t.Errorf("empty WriteFile produced non-empty output: %q", data)
	}
}

func TestWriteFileDelta(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "v")
	dir := t.TempDir()
	path := dir + "/delta.ini"
	if err := cfg.WriteFileDelta(path); err != nil {
		t.Fatalf("WriteFileDelta: %v", err)
	}
	data, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "k=v") {
		t.Errorf("delta output missing k=v: %q", data)
	}
}

// --- io.go edge cases ---

func TestSetFileNameEmpty(t *testing.T) {
	cfg := New()
	if err := cfg.SetFileName(""); !errors.Is(err, ErrEmptyArg) {
		t.Errorf("SetFileName(\"\") = %v, want ErrEmptyArg", err)
	}
}

func TestGetFileNameDefault(t *testing.T) {
	cfg := New()
	if got := cfg.GetFileName(); got != "" {
		t.Errorf("GetFileName() = %q, want empty", got)
	}
}

func TestGetWriteModeDefault(t *testing.T) {
	cfg := New()
	if got := cfg.GetWriteMode(); got != 0 {
		t.Errorf("GetWriteMode() = %o, want 0", got)
	}
}

func TestErrorsEmpty(t *testing.T) {
	cfg := New()
	if errs := cfg.Errors(); errs != nil {
		t.Errorf("Errors() = %v, want nil", errs)
	}
}

func TestReloadNoFile(t *testing.T) {
	cfg := New()
	if err := cfg.Reload(); err != nil {
		t.Errorf("Reload no file = %v, want nil", err)
	}
}

func TestReloadNoFileWithImport(t *testing.T) {
	imp := New()
	imp.NewVal("s", "k", "v")
	cfg := New(WithImport(imp))
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if v, ok := cfg.Val("s", "k"); !ok || v != "v" {
		t.Errorf("after Reload with import Val = (%q,%v), want (v,true)", v, ok)
	}
}

// --- value.go edge cases ---

func TestValueSliceCopy(t *testing.T) {
	v := multiValue([]string{"a", "b"})
	s := v.slice()
	s[0] = "X"
	again := v.slice()
	if again[0] != "a" {
		t.Errorf("slice mutation leaked; again[0] = %q, want a", again[0])
	}
}

func TestSingleValueSlice(t *testing.T) {
	v := singleValue("x")
	s := v.slice()
	if len(s) != 1 || s[0] != "x" {
		t.Errorf("singleValue slice = %v, want [x]", s)
	}
}

// --- parse.go parseLoop edge cases ---

func TestParseLoopMalformedLine(t *testing.T) {
	data := []byte("[s]\nno-equals-here\nk=v\n")
	cfg := New()
	err := cfg.parse(data)
	// Malformed line should produce an error.
	if err == nil {
		t.Errorf("parse with malformed line = nil, want error")
	}
}

func TestParseLoopParamOutsideSection(t *testing.T) {
	data := []byte("k=v\n")
	cfg := New()
	err := cfg.parse(data)
	if err == nil {
		t.Errorf("parse param outside section = nil, want error")
	}
}

func TestParseLoopCommentOnly(t *testing.T) {
	data := []byte("# just a comment\n; another comment\n")
	cfg := New(WithAllowEmpty(true))
	if err := cfg.parse(data); err != nil {
		t.Errorf("parse comment-only = %v, want nil", err)
	}
}

func TestParseLoopBlankOnly(t *testing.T) {
	data := []byte("\n  \n\t\n")
	cfg := New(WithAllowEmpty(true))
	if err := cfg.parse(data); err != nil {
		t.Errorf("parse blank-only = %v, want nil", err)
	}
}

func TestNewSectionWithPendingComments(t *testing.T) {
	data := []byte("# comment before section\n[s]\nk=v\n")
	cfg := New()
	if err := cfg.parse(data); err != nil {
		t.Fatalf("parse: %v", err)
	}
	cs, _ := cfg.GetSectionComment("s")
	if len(cs) != 1 || cs[0] != "# comment before section" {
		t.Errorf("section comment = %v, want [# comment before section]", cs)
	}
}

func TestParamAssignmentFallback(t *testing.T) {
	data := []byte("k=v\n")
	cfg := New(WithFallback("FB"))
	if err := cfg.parse(data); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if v, ok := cfg.Val("FB", "k"); !ok || v != "v" {
		t.Errorf("fallback Val = (%q,%v), want (v,true)", v, ok)
	}
}

func TestParamAssignmentNoFallbackOutsideSection(t *testing.T) {
	data := []byte("k=v\n")
	cfg := New()
	err := cfg.parse(data)
	if err == nil {
		t.Errorf("param outside section without fallback = nil, want error")
	}
}

// --- handleComment negative delta tests ---

func TestHandleCommentSectionDelete(t *testing.T) {
	data := []byte("[s1]\nk=v\n[s2]\nk2=v2\n# [s1] is deleted\n")
	cfg := New(WithNegativeDeltas(true))
	if err := cfg.parse(data); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.SectionExists("s1") {
		t.Errorf("s1 should be deleted by negative delta")
	}
	if !cfg.SectionExists("s2") {
		t.Errorf("s2 should still exist")
	}
}

func TestHandleCommentParamDelete(t *testing.T) {
	data := []byte("[s]\nk1=v1\nk2=v2\n# k1 is deleted\n")
	cfg := New(WithNegativeDeltas(true))
	if err := cfg.parse(data); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Exists("s", "k1") {
		t.Errorf("k1 should be deleted by negative delta")
	}
	if !cfg.Exists("s", "k2") {
		t.Errorf("k2 should still exist")
	}
}

// --- ensureDirStat edge cases ---

func TestEnsureDirStatNoFileMode(t *testing.T) {
	cfg := New()
	// When fileMode is 0 and the path doesn't exist, ensureDirStat is a no-op.
	_ = cfg.ensureDirStat("/no/such/path")
	if cfg.fileMode != 0 {
		t.Errorf("fileMode = %o, want 0 after stat of missing file", cfg.fileMode)
	}
}

func TestEnsureDirStatExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/c.ini"
	if err := writeFile(path, "[s]\nk=v\n", 0640); err != nil {
		t.Fatal(err)
	}
	cfg := New()
	_ = cfg.ensureDirStat(path)
	if cfg.fileMode != 0640 {
		t.Errorf("fileMode = %o, want 0640", cfg.fileMode)
	}
}

func TestEnsureDirStatAlreadySet(t *testing.T) {
	cfg := New()
	cfg.fileMode = 0777
	// When fileMode is already set, ensureDirStat returns immediately.
	_ = cfg.ensureDirStat("/no/such/path")
	if cfg.fileMode != 0777 {
		t.Errorf("fileMode changed to %o, want 0777", cfg.fileMode)
	}
}

// --- model.go edge cases ---

func TestDeepCopySectionPreservesParams(t *testing.T) {
	s := &section{
		name:   "s",
		params: []*param{{name: "k1", val: singleValue("v1")}, {name: "k2", val: singleValue("v2")}},
	}
	dc := deepCopySection(s)
	if dc.name != "s" {
		t.Errorf("deepCopySection name = %q, want s", dc.name)
	}
	if !dc.imported {
		t.Errorf("deepCopySection imported = false, want true")
	}
	if len(dc.params) != 2 {
		t.Fatalf("deepCopySection params = %d, want 2", len(dc.params))
	}
	// Mutating original should not affect copy.
	s.params[0].name = "changed"
	if dc.params[0].name != "k1" {
		t.Errorf("deepCopySection mutation leaked; params[0].name = %q", dc.params[0].name)
	}
}

func TestDeepCopyParamPreservesTrailing(t *testing.T) {
	p := &param{name: "k", val: singleValue("v"), trailing: "trailing comment", eot: "EOT", hasEOT: true}
	dc := deepCopyParam(p)
	if dc.trailing != "trailing comment" {
		t.Errorf("deepCopyParam trailing = %q, want trailing comment", dc.trailing)
	}
	if dc.eot != "EOT" || !dc.hasEOT {
		t.Errorf("deepCopyParam eot = %q, hasEOT = %v", dc.eot, dc.hasEOT)
	}
	if dc.touched {
		t.Errorf("deepCopyParam touched = true, want false")
	}
}

func TestDeepCopyCommentsEmpty(t *testing.T) {
	if got := deepCopyComments(nil); got != nil {
		t.Errorf("deepCopyComments(nil) = %v, want nil", got)
	}
	if got := deepCopyComments([]string{}); got != nil {
		t.Errorf("deepCopyComments([]) = %v, want nil", got)
	}
}

func TestJoinCommentLinesEdge(t *testing.T) {
	got := joinCommentLines([]string{"a", "b", "c"})
	if got != "a\nb\nc" {
		t.Errorf("joinCommentLines = %q, want a\\nb\\nc", got)
	}
	if got := joinCommentLines(nil); got != "" {
		t.Errorf("joinCommentLines(nil) = %q, want empty", got)
	}
}

// --- lines.go edge cases ---

func TestSplitLinesNoTerminator(t *testing.T) {
	lines, lineEnds := splitLines([]byte("no-terminator-here"))
	if lines != nil {
		t.Errorf("splitLines no terminator lines = %v, want nil", lines)
	}
	if lineEnds != nil {
		t.Errorf("splitLines no terminator lineEnds = %v, want nil", lineEnds)
	}
}

func TestSplitLinesCRLF(t *testing.T) {
	lines, lineEnds := splitLines([]byte("[s]\r\nk=v\r\n"))
	if !sliceEq(lines, []string{"[s]", "k=v"}) {
		t.Errorf("splitLines CRLF lines = %v", lines)
	}
	if len(lineEnds) != 2 || lineEnds[0] != 0x0D || lineEnds[1] != 0x0A {
		t.Errorf("splitLines CRLF lineEnds = %v, want [0x0D 0x0A]", lineEnds)
	}
}

func TestSplitLinesCR(t *testing.T) {
	lines, lineEnds := splitLines([]byte("[s]\rk=v\r"))
	if !sliceEq(lines, []string{"[s]", "k=v"}) {
		t.Errorf("splitLines CR lines = %v", lines)
	}
	if len(lineEnds) != 1 || lineEnds[0] != 0x0D {
		t.Errorf("splitLines CR lineEnds = %v, want [0x0D]", lineEnds)
	}
}

func TestSplitLinesX15(t *testing.T) {
	lines, lineEnds := splitLines([]byte("[s]\x15k=v\x15"))
	if !sliceEq(lines, []string{"[s]", "k=v"}) {
		t.Errorf("splitLines \\x15 lines = %v", lines)
	}
	if len(lineEnds) != 1 || lineEnds[0] != 0x15 {
		t.Errorf("splitLines \\x15 lineEnds = %v, want [0x15]", lineEnds)
	}
}

func TestSplitLinesBOM(t *testing.T) {
	data := []byte{0xEF, 0xBB, 0xBF, 'a', '\n', 'b', '\n'}
	lines, _ := splitLines(data)
	if !sliceEq(lines, []string{"a", "b"}) {
		t.Errorf("splitLines BOM lines = %v", lines)
	}
}

func TestSplitLinesCRLFTrailing(t *testing.T) {
	lines, _ := splitLines([]byte("a\r\nb\r\n"))
	if !sliceEq(lines, []string{"a", "b"}) {
		t.Errorf("splitLines CRLF trailing = %v", lines)
	}
}

func TestSplitLinesCRTrailing(t *testing.T) {
	lines, _ := splitLines([]byte("a\rb\r"))
	if !sliceEq(lines, []string{"a", "b"}) {
		t.Errorf("splitLines CR trailing = %v", lines)
	}
}

// --- options.go edge cases ---

func TestValidateCommentCharsZero(t *testing.T) {
	if err := validateCommentChars(0, nil); err == nil {
		t.Errorf("validateCommentChars(0) = nil, want error")
	}
}

func TestValidateCommentCharsBadChar(t *testing.T) {
	if err := validateCommentChars('a', nil); err == nil {
		t.Errorf("validateCommentChars('a') = nil, want error")
	}
}

func TestValidateCommentCharsBadAllowed(t *testing.T) {
	if err := validateCommentChars('#', []rune{'1'}); err == nil {
		t.Errorf("validateCommentChars with digit allowed = nil, want error")
	}
}

func TestValidateCommentCharsGood(t *testing.T) {
	if err := validateCommentChars('#', []rune{';', '!'}); err != nil {
		t.Errorf("validateCommentChars valid = %v, want nil", err)
	}
}

// --- misc integration tests ---

func TestRoundTripScalar(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "hello world")
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	cfg2, err := NewFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	v, ok := cfg2.Val("s", "k")
	if !ok || v != "hello world" {
		t.Errorf("round-trip Val = (%q,%v), want (hello world,true)", v, ok)
	}
}

func TestRoundTripMultiValued(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "line1", "line2", "line3")
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	cfg2, err := NewFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	vs, ok := cfg2.Vals("s", "k")
	if !ok || !sliceEq(vs, []string{"line1", "line2", "line3"}) {
		t.Errorf("round-trip Vals = %v, want [line1 line2 line3]", vs)
	}
}

func TestRoundTripTrailingComment(t *testing.T) {
	cfg := New(WithTrailingComment(true))
	cfg.NewVal("s", "k", "val")
	cfg.SetParameterTrailingComment("s", "k", "my comment")
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	cfg2, err := NewFromBytes(buf.Bytes(), WithTrailingComment(true))
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	tc, ok := cfg2.GetParameterTrailingComment("s", "k")
	if !ok || tc != "my comment" {
		t.Errorf("round-trip trailing = (%q,%v), want (my comment,true)", tc, ok)
	}
}

func TestMultipleSectionsOrder(t *testing.T) {
	cfg := New()
	for _, s := range []string{"c", "a", "b"} {
		cfg.NewVal(s, "k", s+"_v")
	}
	got := cfg.Sections()
	if !sliceEq(got, []string{"c", "a", "b"}) {
		t.Errorf("Sections order = %v, want [c a b]", got)
	}
}

func TestDeleteAndDelta(t *testing.T) {
	cfg := New()
	cfg.NewVal("s1", "k1", "v1")
	cfg.NewVal("s2", "k2", "v2")
	cfg.DeleteSection("s1")

	var buf bytes.Buffer
	if _, err := cfg.WriteToDelta(&buf); err != nil {
		t.Fatalf("WriteToDelta: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[s1] is deleted") {
		t.Errorf("delta missing section deletion;\n%s", out)
	}
	if !strings.Contains(out, "k2=v2") {
		t.Errorf("delta missing live section;\n%s", out)
	}
}

func TestValWithDefaultNotFound(t *testing.T) {
	cfg := New()
	got := cfg.ValWithDefault("s", "k", "fallback")
	if got != "fallback" {
		t.Errorf("ValWithDefault = %q, want fallback", got)
	}
}

func TestValWithDefaultFound(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "actual")
	got := cfg.ValWithDefault("s", "k", "fallback")
	if got != "actual" {
		t.Errorf("ValWithDefault = %q, want actual", got)
	}
}

func TestCopySectionWithGroup(t *testing.T) {
	cfg := New()
	cfg.NewVal("g one", "k", "v1")
	cfg.NewVal("g two", "k", "v2")
	if err := cfg.CopySection("g one", "g three", true); err != nil {
		t.Fatalf("CopySection: %v", err)
	}
	members := cfg.GroupMembers("g")
	if !sliceEq(members, []string{"g one", "g two", "g three"}) {
		t.Errorf("GroupMembers(g) = %v", members)
	}
	if v, ok := cfg.Val("g three", "k"); !ok || v != "v1" {
		t.Errorf("copied section Val = (%q,%v), want (v1,true)", v, ok)
	}
}

func TestCopySectionWithGroupSkipsExisting(t *testing.T) {
	cfg := New()
	cfg.NewVal("g one", "k", "v1")
	cfg.NewVal("g two", "k", "v2")
	// Pre-create "g three" with its own value.
	cfg.NewVal("g three", "k", "original")
	if err := cfg.CopySection("g one", "g four", true); err != nil {
		t.Fatalf("CopySection: %v", err)
	}
	// "g three" was NOT a counterpart of "g one" (different suffix), so it's not skipped.
	// But let's verify "g four" got the params.
	if v, ok := cfg.Val("g four", "k"); !ok || v != "v1" {
		t.Errorf("copied section Val = (%q,%v), want (v1,true)", v, ok)
	}
	// And "g three" retains its original value.
	if v, ok := cfg.Val("g three", "k"); !ok || v != "original" {
		t.Errorf("g three Val = (%q,%v), want (original,true)", v, ok)
	}
	members := cfg.GroupMembers("g")
	if len(members) != 4 {
		t.Errorf("GroupMembers(g) = %v, want 4 members", members)
	}
}

func TestDelete(t *testing.T) {
	cfg := New()
	cfg.NewVal("s1", "k", "v")
	cfg.NewVal("s2", "k", "v")
	cfg.Delete()
	if len(cfg.Sections()) != 0 {
		t.Errorf("Sections after Delete = %v, want []", cfg.Sections())
	}
}

func TestRenameSectionEdge(t *testing.T) {
	cfg := New()
	cfg.NewVal("old", "k", "v")
	if err := cfg.RenameSection("old", "new", false); err != nil {
		t.Fatalf("RenameSection: %v", err)
	}
	if cfg.SectionExists("old") {
		t.Errorf("old section still exists")
	}
	if v, ok := cfg.Val("new", "k"); !ok || v != "v" {
		t.Errorf("Val(new,k) = (%q,%v), want (v,true)", v, ok)
	}
}

func TestRenameSectionWithGroupEdge(t *testing.T) {
	cfg := New()
	cfg.NewVal("g one", "k", "v1")
	cfg.NewVal("g two", "k", "v2")
	if err := cfg.RenameSection("g one", "g three", true); err != nil {
		t.Fatalf("RenameSection: %v", err)
	}
	if cfg.SectionExists("g one") {
		t.Errorf("g one still exists")
	}
	members := cfg.GroupMembers("g")
	if !sliceEq(members, []string{"g two", "g three"}) {
		t.Errorf("GroupMembers after rename = %v", members)
	}
}

func TestWriteFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/c.ini"
	if err := writeFile(path, "[s]\nk=v\n", 0640); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.NewVal("s2", "k2", "v2")
	if err := cfg.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := fileStat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("mode after WriteFile = %o, want 0640", info.Mode().Perm())
	}
}

func TestWriteFileExplicitMode(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/c.ini"
	cfg := New()
	cfg.NewVal("s", "k", "v")
	cfg.SetWriteMode(0600)
	if err := cfg.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := fileStat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestRewriteNoFile(t *testing.T) {
	cfg := New()
	if err := cfg.Rewrite(); err != nil {
		t.Errorf("Rewrite no file = %v, want nil", err)
	}
}

func TestSetWriteModeZero(t *testing.T) {
	cfg := New()
	cfg.SetWriteMode(0644)
	if got := cfg.GetWriteMode(); got != 0644 {
		t.Errorf("GetWriteMode after set = %o, want 0644", got)
	}
	cfg.SetWriteMode(0)
	if got := cfg.GetWriteMode(); got != 0 {
		t.Errorf("GetWriteMode after clear = %o, want 0", got)
	}
}

func TestReloadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/c.ini"
	if err := writeFile(path, "[s]\nk=v1\n", 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Modify in-memory.
	cfg.SetVal("s", "k", "v2")
	if v, _ := cfg.Val("s", "k"); v != "v2" {
		t.Fatalf("SetVal failed: %q", v)
	}
	// Reload from disk.
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if v, _ := cfg.Val("s", "k"); v != "v1" {
		t.Errorf("after Reload Val = %q, want v1", v)
	}
}

func TestReloadWarn(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/c.ini"
	if err := writeFile(path, "[s]\nk=v\n", 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(path, WithReloadWarn(true))
	if err != nil {
		t.Fatal(err)
	}
	// After construction, reloadCnt=1. First Reload increments to 2 (>1), emitting warning.
	// writeReloadWarn defaults to os.Stderr; we just verify Reload returns nil.
	if err := cfg.Reload(); err != nil {
		t.Fatalf("Reload with ReloadWarn: %v", err)
	}
}

func TestSectionCommentRoundTrip(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	cfg.SetSectionComment("s", "# first", "# second")
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# first") || !strings.Contains(out, "# second") {
		t.Errorf("output missing section comments;\n%s", out)
	}
}

func TestDeleteSectionComment(t *testing.T) {
	cfg := New()
	cfg.AddSection("s")
	cfg.SetSectionComment("s", "# comment")
	if err := cfg.DeleteSectionComment("s"); err != nil {
		t.Fatalf("DeleteSectionComment: %v", err)
	}
	cs, _ := cfg.GetSectionComment("s")
	if len(cs) != 0 {
		t.Errorf("GetSectionComment after delete = %v, want []", cs)
	}
}

func TestParameterCommentRoundTrip(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "v")
	cfg.SetParameterComment("s", "k", "# param cmt")
	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "# param cmt") {
		t.Errorf("output missing param comment;\n%s", out)
	}
}

func TestDeleteParameterComment(t *testing.T) {
	cfg := New()
	cfg.NewVal("s", "k", "v")
	cfg.SetParameterComment("s", "k", "# cmt")
	if err := cfg.DeleteParameterComment("s", "k"); err != nil {
		t.Fatalf("DeleteParameterComment: %v", err)
	}
	cs, _ := cfg.GetParameterComment("s", "k")
	if len(cs) != 0 {
		t.Errorf("GetParameterComment after delete = %v, want []", cs)
	}
}

func TestResortSectionsFull(t *testing.T) {
	cfg := New()
	cfg.AddSection("a")
	cfg.AddSection("b")
	cfg.AddSection("c")
	if err := cfg.ResortSections([]string{"c", "b", "a"}); err != nil {
		t.Fatalf("ResortSections: %v", err)
	}
	got := cfg.Sections()
	if !sliceEq(got, []string{"c", "b", "a"}) {
		t.Errorf("full resort = %v, want [c b a]", got)
	}
}

func TestResortSectionsSingle(t *testing.T) {
	cfg := New()
	cfg.AddSection("a")
	cfg.AddSection("b")
	if err := cfg.ResortSections([]string{"a"}); err != nil {
		t.Errorf("ResortSections single = %v, want nil", err)
	}
}

func TestIsImportedPHPCompat(t *testing.T) {
	imp := New(WithPHPCompat(true))
	imp.NewVal("s", "arr[]", "v")
	cfg := New(WithImport(imp), WithPHPCompat(true))
	if !cfg.IsImported("s", "arr[]") {
		t.Errorf("IsImported PHPCompat arr[] = false, want true")
	}
}

func TestGroupMemberSetAndRemove(t *testing.T) {
	cfg := New()
	cfg.AddSection("g one")
	cfg.AddSection("g two")
	members := cfg.GroupMembers("g")
	if len(members) != 2 {
		t.Fatalf("GroupMembers = %v, want 2 members", members)
	}
	cfg.RemoveGroupMember("g one")
	members = cfg.GroupMembers("g")
	if !sliceEq(members, []string{"g two"}) {
		t.Errorf("GroupMembers after remove = %v, want [g two]", members)
	}
}

func TestGroupMembersEmpty(t *testing.T) {
	cfg := New()
	if ms := cfg.GroupMembers("nonexistent"); len(ms) != 0 {
		t.Errorf("GroupMembers(nonexistent) = %v, want []", ms)
	}
}

// helper to read a file back as string
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

// helper to write a file
func writeFile(path, content string, mode os.FileMode) error {
	return os.WriteFile(path, []byte(content), mode)
}

// helper to stat a file
func fileStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

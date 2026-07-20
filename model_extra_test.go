package inifiles

import (
	"testing"
)

// joinCommentLines joins with newline (direct unit test — the helper is
// currently dead code with no callers in production).
func TestJoinCommentLines(t *testing.T) {
	if got := joinCommentLines(nil); got != "" {
		t.Errorf("joinCommentLines(nil) = %q, want empty", got)
	}
	if got := joinCommentLines([]string{"a", "b", "c"}); got != "a\nb\nc" {
		t.Errorf("joinCommentLines = %q, want a\\nb\\nc", got)
	}
	if got := joinCommentLines([]string{"one"}); got != "one" {
		t.Errorf("joinCommentLines([one]) = %q, want one", got)
	}
}

// lineStartsWithComment: whitespace-only line returns false; line whose first
// non-space rune is allowed returns true; not-allowed returns false.
func TestLineStartsWithComment(t *testing.T) {
	allowed := map[rune]struct{}{'#': {}, ';': {}}
	cases := []struct {
		line string
		want bool
	}{
		{"", false},
		{"   ", false},
		{"\t\t", false},
		{"# hi", true},
		{"; hi", true},
		{"  \t# indented", true},
		{"@ nope", false},
		{"x", false},
		{" ! nope", false},
	}
	for _, tc := range cases {
		if got := lineStartsWithComment(tc.line, allowed); got != tc.want {
			t.Errorf("lineStartsWithComment(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

// markupComments prefixes lines lacking an allowed leader, keeps prefixed lines
// verbatim, including when the allowed set differs from the comment char.
func TestMarkupComments(t *testing.T) {
	got := markupComments([]string{"bare line", "# already", "; already", "  # indented"},
		'#', []rune{';'})
	want := []string{"# bare line", "# already", "; already", "  # indented"}
	if !sliceEq(got, want) {
		t.Errorf("markupComments = %v, want %v", got, want)
	}
	// Empty input -> empty result.
	if got := markupComments(nil, '#', nil); len(got) != 0 {
		t.Errorf("markupComments(nil) = %v, want empty", got)
	}
}

// deepCopyComments nil/empty -> nil; otherwise a fresh copy.
func TestDeepCopyComments(t *testing.T) {
	if got := deepCopyComments(nil); got != nil {
		t.Errorf("deepCopyComments(nil) = %v, want nil", got)
	}
	if got := deepCopyComments([]string{}); got != nil {
		t.Errorf("deepCopyComments(empty) = %v, want nil", got)
	}
	in := []string{"a", "b"}
	got := deepCopyComments(in)
	if !sliceEq(got, in) {
		t.Errorf("deepCopyComments = %v, want %v", got, in)
	}
	// Mutating the copy does not affect the original.
	got[0] = "X"
	if in[0] != "a" {
		t.Errorf("deepCopyComments did not copy; original mutated to %q", in[0])
	}
}

// deepCopyParam / deepCopySection produce independent copies; section copy
// marks imported=true and resets touched.
func TestDeepCopyParamAndSection(t *testing.T) {
	s := &section{name: "S", comments: []string{"# c"}, touched: true}
	s.params = append(s.params, &param{name: "k", val: singleValue("v"), touched: true, eot: "EOT", hasEOT: true, trailing: "t", comments: []string{"# pc"}})

	dc := deepCopySection(s)
	if dc.name != "S" || !dc.imported || dc.touched {
		t.Errorf("deepCopySection flags: name=%q imported=%v touched=%v", dc.name, dc.imported, dc.touched)
	}
	if !sliceEq(dc.comments, []string{"# c"}) {
		t.Errorf("deepCopySection comments = %v, want [# c]", dc.comments)
	}
	if len(dc.params) != 1 {
		t.Fatalf("deepCopySection params len = %d, want 1", len(dc.params))
	}
	dp := dc.params[0]
	if dp.name != "k" || dp.touched {
		t.Errorf("deepCopyParam: name=%q touched=%v, want k/false", dp.name, dp.touched)
	}
	if dp.eot != "EOT" || !dp.hasEOT || dp.trailing != "t" {
		t.Errorf("deepCopyParam fields not copied: eot=%q hasEOT=%v trailing=%q", dp.eot, dp.hasEOT, dp.trailing)
	}
	// Mutating copy's value must not affect original.
	dp.val.lines[0] = "X"
	if s.params[0].val.scalar() != "v" {
		t.Errorf("deepCopyParam did not copy value; original = %q", s.params[0].val.scalar())
	}
}

// section.param / paramIndex / addParam / deleteParam / paramNames round-trip.
func TestSection_ParamOps(t *testing.T) {
	var s section
	if s.param("x") != nil {
		t.Errorf("param(x) on empty = non-nil")
	}
	if s.paramIndex("x") != -1 {
		t.Errorf("paramIndex(x) on empty = %d, want -1", s.paramIndex("x"))
	}
	s.addParam(&param{name: "a"})
	s.addParam(&param{name: "b"})
	if s.paramIndex("b") != 1 {
		t.Errorf("paramIndex(b) = %d, want 1", s.paramIndex("b"))
	}
	if got := s.paramNames(); !sliceEq(got, []string{"a", "b"}) {
		t.Errorf("paramNames = %v, want [a b]", got)
	}
	s.deleteParam("a")
	if got := s.paramNames(); !sliceEq(got, []string{"b"}) {
		t.Errorf("paramNames after delete = %v, want [b]", got)
	}
	// deleteParam of a missing name is a no-op.
	s.deleteParam("missing")
	if got := s.paramNames(); !sliceEq(got, []string{"b"}) {
		t.Errorf("paramNames after missing-delete = %v, want [b]", got)
	}
}

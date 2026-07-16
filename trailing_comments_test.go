package inifiles

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Translated from config-inifiles/t/22trailing-comment-lines.t.
func TestTrailingComments_EndCommentsPreserved(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "trailing-comments.ini"))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	var buf strings.Builder
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()

	// qr{; End Comment 1\n; End Comment 2\n+\z}ms
	matched, err := regexp.MatchString(`; End Comment 1\n; End Comment 2\n+\z`, out)
	if err != nil {
		t.Fatalf("regexp error: %v", err)
	}
	if !matched {
		t.Errorf("end comments not preserved; output=\n%q", out)
	}
}

// Translated from config-inifiles/t/29end-of-line-comment.t.
func TestTrailingComments_HandleTrailingComment(t *testing.T) {
	// handle_trailing_comment enabled.
	cfg, err := NewFromFile(testdata(t, "end-of-line-comment.ini"),
		WithTrailingComment(true))
	if err != nil {
		t.Fatalf("NewFromFile enabled: %v", err)
	}

	// value split on ';'.
	if v, ok := cfg.Val("section1", "param1"); !ok || v != "value1" {
		t.Errorf(`Val(section1,param1) = %q ok=%v, want "value1"`, v, ok)
	}

	// trailing comment retrieved.
	tc, _ := cfg.GetParameterTrailingComment("section1", "param1")
	if tc != "comment1" {
		t.Errorf(`GetParameterTrailingComment = %q, want "comment1"`, tc)
	}

	// Write back: output appends " # comment1" after the value.
	var buf strings.Builder
	if _, err := cfg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	matched, _ := regexp.MatchString(`param1\s*=\s*value1\s*[;#]\s*comment1`, buf.String())
	if !matched {
		t.Errorf("trailing comment not rewritten; output=\n%s", buf.String())
	}

	// SetParameterTrailingComment.
	if err := cfg.SetParameterTrailingComment("section1", "param1", "changed comment1"); err != nil {
		t.Fatalf("SetParameterTrailingComment: %v", err)
	}
	tc, _ = cfg.GetParameterTrailingComment("section1", "param1")
	if tc != "changed comment1" {
		t.Errorf(`GetParameterTrailingComment after set = %q, want "changed comment1"`, tc)
	}

	// Use a temp file write+read to exercise WriteConfig path too.
	tmpPath := filepath.Join(t.TempDir(), "end-trailing-comment-writeback.ini")
	if err := cfg.WriteFile(tmpPath); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	matched, _ = regexp.MatchString(`param1\s*=\s*value1\s*[;#]\s*changed comment1`, string(data))
	if !matched {
		t.Errorf("writeback file missing trailing comment; file=\n%s", string(data))
	}
}

// Translated from config-inifiles/t/29end-of-line-comment.t (disabled branch).
func TestTrailingComment_Disabled(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "end-of-line-comment.ini"),
		WithTrailingComment(false))
	if err != nil {
		t.Fatalf("NewFromFile disabled: %v", err)
	}
	if v, ok := cfg.Val("section1", "param1"); !ok || v != "value1;comment1" {
		t.Errorf(`Val(section1,param1) disabled = %q ok=%v, want "value1;comment1"`, v, ok)
	}
	tc, _ := cfg.GetParameterTrailingComment("section1", "param1")
	if tc != "" {
		t.Errorf(`GetParameterTrailingComment disabled = %q, want ""`, tc)
	}
}

// Translated from config-inifiles/t/29end-of-line-comment.t (default branch).
func TestTrailingComment_DefaultOff(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "end-of-line-comment.ini"))
	if err != nil {
		t.Fatalf("NewFromFile default: %v", err)
	}
	if v, ok := cfg.Val("section1", "param1"); !ok || v != "value1;comment1" {
		t.Errorf(`Val(section1,param1) default = %q ok=%v, want "value1;comment1"`, v, ok)
	}
}

// Translated from config-inifiles/t/31comments_with_spaces.t.
func TestTrailingComments_WithSpaces(t *testing.T) {
	iniContents := "[section]\n" +
		"value1 = xxx ; My Comment\n" +
		"value2 = xxx ; My_Comment\n"

	tmpPath := filepath.Join(t.TempDir(), "test31.ini")
	if err := os.WriteFile(tmpPath, []byte(iniContents), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := NewFromFile(tmpPath,
		WithTrailingComment(true),
		WithCommentChar(';'),
		WithAllowedCommentChars([]rune{';', '#'}))
	if err != nil {
		t.Fatalf("NewFromFile: %v", err)
	}

	if v, ok := cfg.Val("section", "value1"); !ok || v != "xxx" {
		t.Errorf(`Val(section,value1) = %q ok=%v, want "xxx"`, v, ok)
	}
	if tc, _ := cfg.GetParameterTrailingComment("section", "value1"); tc != "My Comment" {
		t.Errorf(`trailing(value1) = %q, want "My Comment"`, tc)
	}
	if v, ok := cfg.Val("section", "value2"); !ok || v != "xxx" {
		t.Errorf(`Val(section,value2) = %q ok=%v, want "xxx"`, v, ok)
	}
	if tc, _ := cfg.GetParameterTrailingComment("section", "value2"); tc != "My_Comment" {
		t.Errorf(`trailing(value2) = %q, want "My_Comment"`, tc)
	}
}

// Translated from config-inifiles/t/34trailing-comments-double-delimeter.t.
func TestTrailingComments_DoubleDelimeter(t *testing.T) {
	iniContents := "[MySect]\n" +
		"key=val ; a comment ; further comment\n" +
		"\n"

	cfg, err := NewFromBytes([]byte(iniContents),
		WithTrailingComment(true))
	if err != nil {
		t.Fatalf("NewFromBytes: %v", err)
	}

	if v, ok := cfg.Val("MySect", "key"); !ok || v != "val" {
		t.Errorf(`Val(MySect,key) = %q ok=%v, want "val"`, v, ok)
	}
}

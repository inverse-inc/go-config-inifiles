package inifiles

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Parse errors carry the line number once, via ParseError.Error.
func TestParseErrorLineNumberNotDuplicated(t *testing.T) {
	cases := map[string]struct{ src, want string }{
		"outside section": {"x=1\n", "1: parameter found outside a section"},
		"no end marker":   {"[s]\np=<<EOT\nline\n", `3: no end marker ("EOT") found`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromBytes([]byte(tc.src))
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Reload without a file name must not append imported sections again.
func TestReloadNoFileDoesNotDuplicateImport(t *testing.T) {
	base, err := NewFromBytes([]byte("[a]\nk=1\n[b]\nk=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	c := New(WithImport(base))
	for i := 0; i < 3; i++ {
		if err := c.Reload(); err != nil {
			t.Fatal(err)
		}
	}
	if got := c.Sections(); len(got) != 2 {
		t.Fatalf("Sections() = %v, want [a b]", got)
	}
	var buf bytes.Buffer
	if _, err := c.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "[a]"); n != 1 {
		t.Errorf("[a] written %d times:\n%s", n, buf.String())
	}
}

// Reload without a file name discards in-memory changes and warns when asked.
func TestReloadNoFileResetsAndWarns(t *testing.T) {
	var warn bytes.Buffer
	c := New(WithReloadWarn(true), WithReloadWriter(&warn))
	if err := c.AddSection("tmp"); err != nil {
		t.Fatal(err)
	}
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := c.Sections(); len(got) != 0 {
		t.Errorf("Sections() after Reload = %v, want none", got)
	}
	if !strings.Contains(warn.String(), "reloading") {
		t.Errorf("expected reload warning, got %q", warn.String())
	}
}

// Invalid comment characters are rejected on write, not only on read.
func TestInvalidCommentCharRejectedOnWrite(t *testing.T) {
	cases := map[string]Option{
		"alnum primary":        WithCommentChar('a'),
		"tab allowed":          WithAllowedCommentChars([]rune{'\t'}),
		"nonprintable allowed": WithAllowedCommentChars([]rune{'\x01'}),
	}
	for name, opt := range cases {
		t.Run(name, func(t *testing.T) {
			c := New(opt)
			if err := c.AddSection("s"); err != nil {
				t.Fatal(err)
			}
			if _, err := c.WriteTo(&bytes.Buffer{}); err == nil {
				t.Error("WriteTo: expected error")
			}
			path := filepath.Join(t.TempDir(), "out.ini")
			if err := c.WriteFile(path); err == nil {
				t.Error("WriteFile: expected error")
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("WriteFile left a file behind: %v", err)
			}
		})
	}
}

// Non-printable allowed comment characters are rejected on read too.
func TestNonPrintableAllowedCommentCharRejectedOnRead(t *testing.T) {
	if _, err := NewFromBytes([]byte("[s]\n"), WithAllowedCommentChars([]rune{'\t'})); err == nil {
		t.Error("expected error for tab comment char")
	}
}

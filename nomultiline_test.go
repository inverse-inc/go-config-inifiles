package inifiles

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// translated from config-inifiles/t/28nomultiline.t
func TestNoMultiline(t *testing.T) {
	t.Run("in-memory with NoMultiline writes repeated lines", func(t *testing.T) {
		cfg := New(WithNoMultiline(true))
		if err := cfg.NewVal("s", "k", "a", "b"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		var buf bytes.Buffer
		if _, err := cfg.WriteTo(&buf); err != nil {
			t.Fatalf("WriteTo: %v", err)
		}
		out := buf.String()
		if regexp.MustCompile(`EOT`).MatchString(out) {
			t.Errorf("output should not contain here-doc form; got:\n%s", out)
		}
		if !regexp.MustCompile(`(?m)^k=a$`).MatchString(out) {
			t.Errorf("output should have k=a line; got:\n%s", out)
		}
		if !regexp.MustCompile(`(?m)^k=b$`).MatchString(out) {
			t.Errorf("output should have k=b line; got:\n%s", out)
		}
	})

	t.Run("without NoMultiline writes here-doc form", func(t *testing.T) {
		cfg := New()
		if err := cfg.NewVal("s", "k", "a", "b"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		var buf bytes.Buffer
		if _, err := cfg.WriteTo(&buf); err != nil {
			t.Fatalf("WriteTo: %v", err)
		}
		out := buf.String()
		if !regexp.MustCompile(`EOT`).MatchString(out) {
			t.Errorf("output should contain here-doc form; got:\n%s", out)
		}
	})

	t.Run("file-based rewrite omits here-doc", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "foo.ini")
		content := "[section]\na = 1\na = 2\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		cfg, err := NewFromFile(path, WithNoMultiline(true))
		if err != nil {
			t.Fatalf("NewFromFile: %v", err)
		}
		if err := cfg.Rewrite(); err != nil {
			t.Fatalf("Rewrite: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		s := string(got)
		if regexp.MustCompile(`EOT`).MatchString(s) {
			t.Errorf("rewritten file should not contain EOT; got:\n%s", s)
		}
		if !regexp.MustCompile(`(?m)^a=1$`).MatchString(s) {
			t.Errorf("rewritten file should have a=1; got:\n%s", s)
		}
		if !regexp.MustCompile(`(?m)^a=2$`).MatchString(s) {
			t.Errorf("rewritten file should have a=2; got:\n%s", s)
		}
	})
}

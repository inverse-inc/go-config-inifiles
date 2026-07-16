package inifiles

import (
	"bytes"
	"strings"
	"testing"
)

// translated from config-inifiles/t/25line-endings.t
//
// A config built in-test is written with each of several line endings
// ("\r\n", "\r", "\n", "\x15"). For each: load via NewFromBytes, assert the
// single and multi (here-doc) values parse correctly, AND assert that writing
// the config back out reproduces the SAME line ending that was detected on
// input.
func TestLineEndings_RoundTrip(t *testing.T) {
	// The logical lines of the sample ini (mirrors the Perl t/25 template
	// after <eol>...<sol> substitution). The first line is intentionally
	// empty so the very first byte is the line terminator (driving detection).
	lines := []string{
		"",
		"# this is a sample file for testing the proper detection of line endings in Config::IniFiles",
		"",
		"[single values]",
		"firstval = first value",
		"secondval=2nd",
		"",
		"# in v2.52 on linux multi values with crlf lines are failing",
		"[multi value]",
		"Paths=<<EOT",
		"path1",
		"path2",
		"EOT",
		"",
		"",
	}

	cases := []struct {
		name string
		sep  string
	}{
		{"CRLF", "\r\n"},
		{"CR", "\r"},
		{"LF", "\n"},
		{"0x15", "\x15"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(strings.Join(lines, tc.sep))
			cfg, err := NewFromBytes(data)
			if err != nil {
				t.Fatalf("NewFromBytes (%q): %v", tc.sep, err)
			}

			// Detected line ending must match the input separator.
			if got := cfg.ors(); got != tc.sep {
				t.Errorf("ors() = %q, want %q", got, tc.sep)
			}

			// Single values.
			if v, ok := cfg.Val("single values", "firstval"); !ok || v != "first value" {
				t.Errorf(`Val(single values,firstval) = %q ok=%v, want "first value"`, v, ok)
			}
			if v, ok := cfg.Val("single values", "secondval"); !ok || v != "2nd" {
				t.Errorf(`Val(single values,secondval) = %q ok=%v, want "2nd"`, v, ok)
			}

			// Multi (here-doc) value.
			vs, ok := cfg.Vals("multi value", "Paths")
			if !ok || len(vs) != 2 || vs[0] != "path1" || vs[1] != "path2" {
				t.Errorf(`Vals(multi value,Paths) = %v ok=%v, want [path1 path2]`, vs, ok)
			}

			// Write back and assert the same line ending is reproduced.
			var buf bytes.Buffer
			if _, err := cfg.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			out := buf.Bytes()
			if !bytes.Contains(out, []byte(tc.sep)) {
				t.Errorf("output does not contain expected line ending %q; output=%q", tc.sep, out)
			}
			// The output, when split on the input separator, must contain the
			// expected logical lines as pieces (proving that separator is what
			// the writer used between lines).
			for _, want := range []string{"firstval=first value", "path1", "path2", "EOT", "[single values]"} {
				if !piecePresent(string(out), tc.sep, want) {
					t.Errorf("output split on %q missing piece %q; output=%q", tc.sep, want, out)
				}
			}
			// No alien line endings: exclude the other terminators.
			for _, alien := range []string{"\r\n", "\r", "\n", "\x15"} {
				if alien == tc.sep {
					continue
				}
				// "\r" vs "\r\n": "\r\n" contains "\r", so only assert absence
				// of an alien terminator that is NOT a substring of our sep.
				if strings.Contains(tc.sep, alien) {
					continue
				}
				if bytes.Contains(out, []byte(alien)) {
					t.Errorf("output contains alien line ending %q (sep=%q); output=%q", alien, tc.sep, out)
				}
			}
		})
	}
}

// piecePresent reports whether want appears as an exact element when s is split
// on sep.
func piecePresent(s, sep, want string) bool {
	for _, p := range strings.Split(s, sep) {
		if p == want {
			return true
		}
	}
	return false
}

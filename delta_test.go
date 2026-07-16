package inifiles

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// stripWhitespace removes every whitespace byte from s (mirrors Perl
// $text =~ s/\s*//g applied to the whole file).
func stripWhitespace(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			continue
		}
		b = append(b, s[i])
	}
	return string(b)
}

func mustSlurp(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// translated from config-inifiles/t/10delta.t
func TestDelta(t *testing.T) {
	en, err := NewFromFile(testdata(t, "en.ini"))
	if err != nil {
		t.Fatalf("NewFromFile en.ini: %v", err)
	}

	es, err := NewFromFile(testdata(t, "es.ini"), WithImport(en))
	if err != nil {
		t.Fatalf("NewFromFile es.ini: %v", err)
	}

	estext := stripWhitespace(mustSlurp(t, testdata(t, "es.ini")))

	// test 3: delta with no changes equals es.ini (whitespace-stripped).
	deltaPath := filepath.Join(t.TempDir(), "delta.ini")
	if err := es.WriteFileDelta(deltaPath); err != nil {
		t.Fatalf("WriteFileDelta: %v", err)
	}
	deltatext := stripWhitespace(mustSlurp(t, deltaPath))
	if deltatext != estext {
		d := deltatext
		if len(d) > 200 {
			d = d[:200]
		}
		e := estext
		if len(e) > 200 {
			e = e[:200]
		}
		t.Errorf("delta without changes != es.ini\nfirst200 delta:\n%s\nfirst200 es:\n%s", d, e)
	}

	// test 4: delta with an update.
	if err := es.NewVal("something", "completely", "different"); err != nil {
		t.Fatalf("NewVal: %v", err)
	}
	if err := es.WriteFileDelta(deltaPath); err != nil {
		t.Fatalf("WriteFileDelta: %v", err)
	}
	deltatext = mustSlurp(t, deltaPath)
	if !regexp.MustCompile(`(?m)^\[something\]`).MatchString(deltatext) {
		t.Errorf("delta should contain [something]; got:\n%s", deltatext)
	}
	if !strings.Contains(deltatext, "completely=different") {
		t.Errorf("delta should contain completely=different; got:\n%s", deltatext)
	}

	// test 5: delta with deletion marks.
	if err := es.DelVal("x", "LongName"); err != nil {
		t.Fatalf("DelVal: %v", err)
	}
	if err := es.DeleteSection("m"); err != nil {
		t.Fatalf("DeleteSection: %v", err)
	}
	var buf bytes.Buffer
	if _, err := es.WriteToDelta(&buf); err != nil {
		t.Fatalf("WriteToDelta: %v", err)
	}
	deltatext = buf.String()
	if !regexp.MustCompile(`(?m)^# \[m\] is deleted`).MatchString(deltatext) {
		t.Errorf("delta should mark [m] deleted; got:\n%s", deltatext)
	}
	if !regexp.MustCompile(`(?m)^# LongName is deleted`).MatchString(deltatext) {
		t.Errorf("delta should mark LongName deleted; got:\n%s", deltatext)
	}

	// test 6: re-reading the delta with import applies deletions.
	if err := es.WriteFileDelta(deltaPath); err != nil {
		t.Fatalf("WriteFileDelta: %v", err)
	}
	// NOTE: NegativeDeltas is required to parse "# X is deleted" markers.
	// (Perl auto-enables it when -import is given; this port needs it
	// explicit because newConfig inspects o.Import before applying options.)
	es2, err := NewFromFile(deltaPath, WithImport(en), WithNegativeDeltas(true))
	if err != nil {
		t.Fatalf("NewFromFile delta: %v", err)
	}
	if _, ok := es2.Val("x", "LongName"); ok {
		t.Errorf("Val(x,LongName) should be deleted/not-found after re-read")
	}
	if es2.SectionExists("m") {
		t.Errorf("SectionExists(m) should be false after re-read")
	}
}

package inifiles

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// translated from config-inifiles/t/15store-and-retrieve-here-doc-terminator.t
//
// A here-doc value whose body contains the default terminator text ("EOT")
// must round-trip losslessly: the writer picks a distinct output terminator
// so the value is preserved verbatim on reload. Run the scenario twice, once
// via WriteTo + NewFromBytes, once via WriteFile + NewFromFile.
func TestHereDocTerminator_RoundTrip(t *testing.T) {
	const want = "Hello\nEOT\n"

	roundTripViaBuffer := func(t *testing.T) {
		cfg := New()
		if err := cfg.NewVal("MySection", "MyParam", "Hello\nEOT\n"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		var buf bytes.Buffer
		if _, err := cfg.WriteTo(&buf); err != nil {
			t.Fatalf("WriteTo: %v", err)
		}
		// The output terminator must differ from a substring of the value;
		// otherwise the reload would truncate the body.
		if strings.Contains(buf.String(), "Hello\nEOT\nEOT\n") {
			// naive "EOT" terminator would collide; ensure writer extended it.
			t.Fatalf("output appears to use a colliding terminator:\n%q", buf.String())
		}

		cfg2, err := NewFromBytes(buf.Bytes())
		if err != nil {
			t.Fatalf("NewFromBytes: %v", err)
		}
		got, ok := cfg2.Val("MySection", "MyParam")
		if !ok || got != want {
			t.Fatalf("Val = %q ok=%v, want %q", got, ok, want)
		}
	}

	roundTripViaFile := func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "store-and-retrieve-here-doc-terminator.ini")

		cfg := New()
		if err := cfg.NewVal("MySection", "MyParam", "Hello\nEOT\n"); err != nil {
			t.Fatalf("NewVal: %v", err)
		}
		if err := cfg.WriteFile(path); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		cfg2, err := NewFromFile(path)
		if err != nil {
			t.Fatalf("NewFromFile: %v", err)
		}
		got, ok := cfg2.Val("MySection", "MyParam")
		if !ok || got != want {
			t.Fatalf("Val = %q ok=%v, want %q", got, ok, want)
		}
	}

	t.Run("WriteTo_NewFromBytes", roundTripViaBuffer)
	t.Run("WriteFile_NewFromFile", roundTripViaFile)
}

// translated from config-inifiles/t/37here-doc-no-end-marker.t
//
// A here-doc whose terminator marker never appears in the body must fail to
// load. NewFromFile returns (nil, err); the error message must mention both
// the phrase "no end marker" and the missing marker token ("NOEND").
func TestHereDocNoEndMarker_Fails(t *testing.T) {
	cfg, err := NewFromFile(testdata(t, "here-doc-no-end-marker.ini"))
	if cfg != nil {
		t.Fatalf("NewFromFile returned non-nil config for unterminated here-doc: %v", cfg)
	}
	if err == nil {
		t.Fatalf("NewFromFile should error on unterminated here-doc")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no end marker") {
		t.Errorf("error %q does not contain \"no end marker\"", msg)
	}
	if !strings.Contains(msg, `"NOEND"`) {
		t.Errorf("error %q does not contain \"NOEND\" in quotes", msg)
	}
}

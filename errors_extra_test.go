package inifiles

import (
	"errors"
	"testing"
)

// ParseError.Error formats with the line number when > 0, else just the
// message (covers both branches).
func TestParseError_ErrorFormatting(t *testing.T) {
	pe := &ParseError{Line: 0, Message: "oops"}
	if got := pe.Error(); got != "oops" {
		t.Errorf("ParseError.Error() line0 = %q, want oops", got)
	}
	pe = &ParseError{Line: 12, Message: "bad line"}
	if got := pe.Error(); got != "12: bad line" {
		t.Errorf("ParseError.Error() line12 = %q, want 12: bad line", got)
	}
}

// ParseError.Unwrap returns nil (covers the 0%-covered method), so errors.Is
// against any sentinel reports false.
func TestParseError_UnwrapIsNil(t *testing.T) {
	pe := &ParseError{Line: 1, Message: "x"}
	if errors.Is(pe, ErrNotFound) {
		t.Errorf("errors.Is(ParseError, ErrNotFound) = true, want false")
	}
	if pe.Unwrap() != nil {
		t.Errorf("ParseError.Unwrap() = %v, want nil", pe.Unwrap())
	}
}

// multiError.Error joins errors with "; " and reports "no errors" when empty.
func TestMultiError_Error(t *testing.T) {
	m := &multiError{errs: nil}
	if got := m.Error(); got != "no errors" {
		t.Errorf("empty multiError.Error() = %q, want no errors", got)
	}
	m = &multiError{errs: []ParseError{
		{Line: 2, Message: "a"},
		{Line: 0, Message: "b"},
	}}
	if got := m.Error(); got != "2: a; b" {
		t.Errorf("multiError.Error() = %q, want 2: a; b", got)
	}
	// Errors() returns the stored slice copy.
	if got := m.Errors(); len(got) != 2 {
		t.Errorf("multiError.Errors() len = %d, want 2", len(got))
	}
}

// errorList helpers: add/addf/reset/has/asError round-trip.
func TestErrorList_Helpers(t *testing.T) {
	var l errorList
	if l.has() {
		t.Errorf("empty errorList.has() = true, want false")
	}
	l.add(3, "three")
	l.addf(4, "four %d", 4)
	if !l.has() {
		t.Errorf("errorList.has() = false after adds")
	}
	err := l.asError()
	if err == nil {
		t.Fatalf("asError = nil, want error")
	}
	if got := err.Error(); got != "3: three; 4: four 4" {
		t.Errorf("asError.Error() = %q, want 3: three; 4: four 4", got)
	}
	l.reset()
	if l.has() {
		t.Errorf("errorList.has() = true after reset")
	}
	if err := l.asError(); err != nil {
		t.Errorf("asError after reset = %v, want nil", err)
	}
}

// A Config with accumulated parse errors is returned with a populated
// Errors() slice. Because parse() surfaces any accumulated errors as a return
// value (yielding a nil config from the constructors), the populated branch of
// Errors() is reached here by adding errors directly to a live (parsed) config.
func TestConfigErrors_Populated(t *testing.T) {
	cfg := New()
	cfg.errs.add(1, "first")
	cfg.errs.add(2, "second")
	got := cfg.Errors()
	if len(got) != 2 {
		t.Fatalf("Errors() len = %d, want 2", len(got))
	}
	if got[0].Line != 1 || got[0].Message != "first" {
		t.Errorf("Errors()[0] = %+v, want Line=1 Message=first", got[0])
	}
	// Errors() returns a copy: mutating it must not affect the config.
	got[1].Message = "mutated"
	again := cfg.Errors()
	if again[1].Message != "second" {
		t.Errorf("Errors() returned non-copy; second call got %q", again[1].Message)
	}
}

// A malformed line causes NewFromBytes to return an error (parse surfaces
// accumulated errs via asError).
func TestNewFromBytes_MalformedReturnsError(t *testing.T) {
	cfg, err := NewFromBytes([]byte("[S]\nbad line without equals\n"))
	if err == nil {
		t.Fatalf("NewFromBytes malformed: err=nil, want error")
	}
	if cfg != nil {
		t.Errorf("NewFromBytes malformed: cfg non-nil, want nil")
	}
}

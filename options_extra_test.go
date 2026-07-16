package inifiles

import (
	"testing"
)

// validateCommentChars rejects a zero comment char (the ch == 0 branch) and
// alphanumeric/[]:= chars for both CommentChar and AllowedCommentChars.
func TestValidateCommentChars_Rejections(t *testing.T) {
	// ch == 0
	if err := validateCommentChars(0, []rune{';'}); err == nil {
		t.Errorf("validateCommentChars(0) = nil, want error")
	}
	// CommentChar is alphanumeric
	for _, ch := range []rune{'a', 'Z', '7', '[', ']', ':', '='} {
		if err := validateCommentChars(ch, nil); err == nil {
			t.Errorf("validateCommentChars(%q) = nil, want error", ch)
		}
	}
	// AllowedCommentChars containing a bad rune is rejected.
	for _, bad := range []rune{'a', '5', '[', ']', ':', '='} {
		if err := validateCommentChars('#', []rune{bad}); err == nil {
			t.Errorf("validateCommentChars('#', [%q]) = nil, want error", bad)
		}
	}
	// Valid combination passes.
	if err := validateCommentChars('#', []rune{';', '@'}); err != nil {
		t.Errorf("validateCommentChars('#',[;@]) = %v, want nil", err)
	}
	// A non-comment rune in allowed that is still a valid punctuation is OK.
	if err := validateCommentChars(';', []rune{'#', '!'}); err != nil {
		t.Errorf("validateCommentChars(';',[#!]) = %v, want nil", err)
	}
}

// Constructors invoking parse surface a validateCommentChars error via the
// parse-time guard when given an illegal comment char.
func TestNewFromBytes_BadCommentChar(t *testing.T) {
	if _, err := NewFromBytes([]byte("[S]\nk=v\n"), WithCommentChar('[')); err == nil {
		t.Errorf("NewFromBytes with CommentChar '[' = nil, want error")
	}
	if _, err := NewFromBytes([]byte("[S]\nk=v\n"), WithAllowedCommentChars([]rune{'='})); err == nil {
		t.Errorf("NewFromBytes with AllowedCommentChars '=' = nil, want error")
	}
}

// WithNegativeDeltas sets the explicit flag so an Import does not auto-enable
// shadowing of a user's explicit Disable.
func TestWithNegativeDeltas_Explicit(t *testing.T) {
	cfg := New(WithNegativeDeltas(false))
	if cfg.opts.NegativeDeltas {
		t.Errorf("NegativeDeltas = true, want false")
	}
	if !cfg.opts.ndExplicit {
		t.Errorf("ndExplicit = false after WithNegativeDeltas, want true")
	}
}

// WithImport auto-enables NegativeDeltas when not set explicitly.
func TestWithImport_AutoEnablesNegativeDeltas(t *testing.T) {
	base := New()
	cfg := New(WithImport(base))
	if !cfg.opts.NegativeDeltas {
		t.Errorf("NegativeDeltas = false after WithImport, want true (auto-enabled)")
	}
	if cfg.opts.ndExplicit {
		t.Errorf("ndExplicit = true after WithImport, want false (not explicit)")
	}
}

// WithReloadWarn and WithReloadWriter set their fields (covered indirectly by
// io_extra; assert directly for completeness).
func TestOptions_Setters(t *testing.T) {
	cfg := New(WithReloadWarn(true), WithNoMultiline(true), WithAllowContinue(true),
		WithAllowEmpty(true), WithNoCase(true), WithPHPCompat(true), WithTrailingComment(true),
		WithDefault("d"), WithFallback("f"))
	o := cfg.opts
	if !o.ReloadWarn || !o.NoMultiline || !o.AllowContinue || !o.AllowEmpty ||
		!o.NoCase || !o.PHPCompat || !o.HandleTrailingComment {
		t.Errorf("option flags not set: %+v", o)
	}
	if o.Default != "d" || o.Fallback != "f" {
		t.Errorf("Default/Fallback = %q/%q, want d/f", o.Default, o.Fallback)
	}
	// caseified defaults under NoCase.
	if cfg.defaultSect != "d" || cfg.fallback != "f" {
		t.Errorf("defaultSect/fallback = %q/%q, want d/f", cfg.defaultSect, cfg.fallback)
	}
}

// WithDefault / WithFallback under NoCase lowercase the stored section name.
func TestOptions_DefaultNoCase(t *testing.T) {
	cfg := New(WithNoCase(true), WithDefault("Defaults"), WithFallback("Fallback"))
	if cfg.defaultSect != "defaults" {
		t.Errorf("defaultSect = %q, want defaults", cfg.defaultSect)
	}
	if cfg.fallback != "fallback" {
		t.Errorf("fallback = %q, want fallback", cfg.fallback)
	}
}

package inifiles

import (
	"errors"
	"fmt"
	"io"
	"unicode"
)

// Options controls parsing and writing behavior, mirroring the Perl
// Config::IniFiles constructor flags.
type Options struct {
	// Import is an existing *Config whose data is deep-copied as a base
	// before reading the local file (local values shadow imported ones).
	Import *Config

	// Default is the section name consulted by Val when a parameter is
	// missing in the requested section. The section is created after every
	// read if the file does not contain it, so it is also written out.
	Default string

	// Fallback is the section into which parameters appearing before any
	// section header are filed. Without it, such parameters are a parse
	// error. The fallback section header is suppressed on write when used.
	Fallback string

	// NoCase lowercases section and parameter keys (values keep original
	// case).
	NoCase bool

	// ReloadWarn prints a reload notice to ReloadWriter (default os.Stderr)
	// on each Reload after the first.
	ReloadWarn bool

	// NoMultiline outputs multi-valued parameters as repeated single lines
	// instead of here-doc blocks. Affects output only.
	NoMultiline bool

	// AllowContinue honors a trailing backslash as a line-continuation
	// marker on read.
	AllowContinue bool

	// AllowEmpty permits config files that contain no sections (including
	// files with only comments or blank lines); otherwise they are an error.
	// The check is made after imported sections are applied, so a config
	// with Import set is only "empty" if the import has no sections either.
	AllowEmpty bool

	// NegativeDeltas parses "; [sect] is deleted" / "; parm is deleted"
	// comments as deletions during read. Auto-enabled when Import is set.
	NegativeDeltas bool

	// CommentChar is the primary comment character used to prefix synthetic
	// comment lines. Must be a printable, non-alphanumeric rune other than
	// [ ] : =. Invalid values are reported as errors when reading or writing.
	CommentChar rune

	// AllowedCommentChars are extra runes recognized as comment leaders in
	// addition to CommentChar. Each must be printable and not alphanumeric
	// or one of [ ] : =. Invalid values are reported as errors when reading
	// or writing.
	AllowedCommentChars []rune

	// HandleTrailingComment parses "value ; comment" into value plus a
	// trailing comment slot.
	HandleTrailingComment bool

	// PHPCompat strips trailing "[]" from parameter names and unwraps
	// quotes from values on read. The "[]" suffix is also stripped from
	// parameter names passed to lookup and mutation methods (Val, SetVal,
	// etc.), so "arr[]" and "arr" refer to the same parameter.
	PHPCompat bool

	// ReloadWriter receives the reload warning message when ReloadWarn is
	// true. Defaults to os.Stderr.
	ReloadWriter io.Writer

	// ndExplicit records whether NegativeDeltas was set by an explicit
	// option (used to avoid auto-enabling-shadowing a user override).
	ndExplicit bool
}

func defaultOptions() Options {
	return Options{
		CommentChar:         '#',
		AllowedCommentChars: []rune{';'},
	}
}

// Option configures Options.
type Option func(*Options)

// WithImport sets an existing *Config whose data is deep-copied as a base
// before reading the local file; local values shadow imported ones.
func WithImport(c *Config) Option { return func(o *Options) { o.Import = c } }

// WithDefault sets the section name consulted by Val when a parameter is
// missing in the requested section. The section is created after every read
// if the file does not contain it, so it also appears in Sections and in
// written output.
func WithDefault(name string) Option { return func(o *Options) { o.Default = name } }

// WithFallback sets the section into which parameters appearing before any
// section header are filed.
func WithFallback(name string) Option { return func(o *Options) { o.Fallback = name } }

// WithNoCase controls whether section and parameter names are lowercased.
// Names are stored lowercased (and written that way), and names passed to
// lookups are lowercased too, so matching is case-insensitive. Values keep
// their original case.
func WithNoCase(b bool) Option { return func(o *Options) { o.NoCase = b } }

// WithReloadWarn controls whether a reload notice is printed on each Reload
// after the first.
func WithReloadWarn(b bool) Option { return func(o *Options) { o.ReloadWarn = b } }

// WithNoMultiline controls whether multi-valued parameters are serialized as
// repeated single lines instead of here-doc blocks.
func WithNoMultiline(b bool) Option { return func(o *Options) { o.NoMultiline = b } }

// WithAllowContinue controls whether a trailing backslash acts as a
// line-continuation marker on read.
func WithAllowContinue(b bool) Option { return func(o *Options) { o.AllowContinue = b } }

// WithAllowEmpty controls whether a config file with no sections is accepted
// or treated as an error. A file containing only comments or blank lines has
// no sections and counts as empty.
func WithAllowEmpty(b bool) Option { return func(o *Options) { o.AllowEmpty = b } }

// WithNegativeDeltas controls whether "; [sect] is deleted" / "; parm is
// deleted" comments trigger deletions during read. This is enabled
// automatically when WithImport is used; pass WithNegativeDeltas(false) to
// turn it off.
func WithNegativeDeltas(b bool) Option {
	return func(o *Options) { o.NegativeDeltas = b; o.ndExplicit = true }
}

// WithCommentChar sets the primary comment character used to prefix synthetic
// comment lines. Must be a printable, non-alphanumeric rune other than
// [ ] : =. Invalid values are reported as errors when reading or writing.
func WithCommentChar(r rune) Option { return func(o *Options) { o.CommentChar = r } }

// WithAllowedCommentChars sets extra runes recognized as comment leaders in
// addition to CommentChar. Each must be a printable, non-alphanumeric rune
// other than [ ] : =. Invalid values are reported as errors when reading or
// writing.
func WithAllowedCommentChars(rs []rune) Option {
	return func(o *Options) { o.AllowedCommentChars = rs }
}

// WithTrailingComment controls whether "value ; comment" is parsed into a
// value plus a trailing comment slot.
func WithTrailingComment(b bool) Option {
	return func(o *Options) { o.HandleTrailingComment = b }
}

// WithPHPCompat controls PHP-style "name[]" array-stripping and quote
// unwrapping on read. When enabled, a trailing "[]" is also stripped from
// parameter names passed to lookup and mutation methods (Val, SetVal, etc.).
func WithPHPCompat(b bool) Option { return func(o *Options) { o.PHPCompat = b } }

// WithReloadWriter sets the writer that receives reload warning messages.
// Defaults to os.Stderr.
func WithReloadWriter(w io.Writer) Option { return func(o *Options) { o.ReloadWriter = w } }

// validateCommentChars returns an error if ch or any rune in allowed is
// invalid (non-printable, whitespace, alphanumeric, or one of [ ] : =),
// matching the Perl module's restrictions.
func validateCommentChars(ch rune, allowed []rune) error {
	bad := func(r rune) bool {
		return !unicode.IsPrint(r) || unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsDigit(r) ||
			r == '[' || r == ']' || r == ':' || r == '='
	}
	if ch == 0 {
		return errors.New("comment character must be unique")
	}
	if bad(ch) {
		return fmt.Errorf("illegal comment character %q", ch)
	}
	for _, r := range allowed {
		if bad(r) {
			return fmt.Errorf("illegal value for allowed comment characters: %q", r)
		}
	}
	return nil
}

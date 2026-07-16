package inifiles

import (
	"errors"
	"os"
)

// Config is a parsed .ini configuration, ported from Perl's
// Config::IniFiles. Use New, NewFromFile, NewFromReader, or NewFromBytes to
// create one.
type Config struct {
	opts     Options
	startup  Options // original options (minus Import) re-applied on Reload
	imported *Config

	file      string // path, "" if none
	fileMode  os.FileMode
	lineEnds  []byte // detected line ending; reused on write
	loaded    bool   // whether a read has been performed
	reloadCnt int

	errs         errorList
	sections     []*section
	sectIdx      map[string]*section
	defaultSect  string
	fallback     string
	fallbackUsed bool
	endComments  []string

	groups     map[string][]string
	groupOrder []string

	mySects []*section // touched sections, ordered, retaining deleted tombstones
}

// NewFromFile creates a Config by reading the file at path.
func NewFromFile(path string, opts ...Option) (*Config, error) {
	c := newConfig(opts...)
	c.file = path
	if err := c.Reload(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewFromReader creates a Config by reading from r. No file path is recorded,
// so WriteFile will require SetFileName first. The detected line ending (if
// any) is reused for subsequent writes. r is consumed entirely.
func NewFromReader(r interface{ Read([]byte) (int, error) }, opts ...Option) (*Config, error) {
	c := newConfig(opts...)
	if err := c.readFromReader(r); err != nil {
		return nil, err
	}
	return c, nil
}

// NewFromBytes creates a Config from an in-memory byte slice. Convenience over
// NewFromReader(bytes.NewReader(b)).
func NewFromBytes(b []byte, opts ...Option) (*Config, error) {
	c := newConfig(opts...)
	if err := c.readFromBytes(b); err != nil {
		return nil, err
	}
	return c, nil
}

// New creates an empty Config without reading any file. Equivalent to Perl
// new() with no -file. Options may include WithFileName via... (use
// SetFileName after) -- callers pass opts for nocase/comment/etc.
func New(opts ...Option) *Config {
	c := newConfig(opts...)
	// Perl's new() always calls ReadConfig, which applies the import (and, with
	// no file, returns immediately). Apply that import eagerly so a file-less
	// config is usable right after construction.
	c.resetState()
	c.applyImport()
	c.reloadCnt = 1
	return c
}

func newConfig(opts ...Option) *Config {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	// Mirror Perl: importing an object auto-enables NegativeDeltas unless the
	// caller passed an explicit WithNegativeDeltas option.
	if o.Import != nil && !o.ndExplicit {
		o.NegativeDeltas = true
	}
	return &Config{
		opts:        o,
		startup:     o,
		imported:    o.Import,
		sectIdx:     map[string]*section{},
		defaultSect: caseifiedName(o.Default, o.NoCase),
		fallback:    caseifiedName(o.Fallback, o.NoCase),
		groups:      map[string][]string{},
	}
}

var errNotImplemented = errors.New("inifiles: not implemented")

// caseify normalizes a section or parameter name per NoCase.
func (c *Config) caseify(name string) string {
	return caseifiedName(name, c.opts.NoCase)
}

func caseifiedName(name string, nocase bool) string {
	if nocase {
		return toLower(name)
	}
	return name
}

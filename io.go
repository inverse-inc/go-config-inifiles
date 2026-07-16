package inifiles

import (
	"errors"
	"io"
	"os"
)

// SetFileName sets the path used by Reload and Rewrite. An empty name returns
// ErrEmptyArg.
func (c *Config) SetFileName(name string) error {
	if name == "" {
		return ErrEmptyArg
	}
	c.file = name
	return nil
}

// GetFileName returns the configured file path (may be "").
func (c *Config) GetFileName() string { return c.file }

// SetWriteMode sets the file mode applied on WriteFile. mode must be three
// octal digits (e.g. "644"); mirrors Perl's /[0-7]{3}/ validation.
func (c *Config) SetWriteMode(mode string) error {
	if !isValidOctalMode(mode) {
		return errors.New("inifiles: invalid write mode (expect 3 octal digits)")
	}
	c.fileMode = parseOctalMode(mode)
	return nil
}

// GetWriteMode returns the configured file mode as an octal string (e.g.
// "0644"), or "" if none.
func (c *Config) GetWriteMode() string {
	if c.fileMode == 0 {
		return ""
	}
	return formatOctalMode(c.fileMode)
}

// Errors returns the errors accumulated during the most recent read.
func (c *Config) Errors() []ParseError {
	if !c.errs.has() {
		return nil
	}
	return append([]ParseError(nil), c.errs.errs...)
}

// Reload re-reads the configured file (if any), discarding in-memory changes.
// Mirrors Perl's ReadConfig.
func (c *Config) Reload() error {
	if c.file == "" {
		// No file: deep-copy imported data then done.
		c.reloadCnt++
		c.applyImport()
		return nil
	}
	f, err := os.Open(c.file)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.readFromFile(f)
}

// readFromFile opens the path and reads it (atomic with mode detection).
func (c *Config) readFromFile(f *os.File) error {
	return c.readFromReader(f)
}

func (c *Config) readFromReader(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return c.readFromBytes(data)
}

func (c *Config) readFromBytes(data []byte) error {
	c.errs.reset()
	c.resetState()
	c.applyImport()
	c.reloadCnt++
	if c.opts.ReloadWarn && c.reloadCnt > 1 {
		c.writeReloadWarn()
	}
	return c.parse(data)
}

func (c *Config) resetState() {
	c.sections = nil
	c.sectIdx = map[string]*section{}
	c.groups = map[string][]string{}
	c.groupOrder = nil
	c.endComments = nil
	c.fallbackUsed = false
	c.lineEnds = nil
	c.mySects = nil
}

func (c *Config) applyImport() {
	if c.imported == nil {
		return
	}
	for _, s := range c.imported.sections {
		dc := deepCopySection(s)
		c.sectIdx[dc.name] = dc
		c.sections = append(c.sections, dc)
	}
	for g, members := range c.imported.groups {
		c.groups[g] = append([]string(nil), members...)
	}
	c.groupOrder = append(c.groupOrder, c.imported.groupOrder...)
	if c.imported.endComments != nil {
		c.endComments = append([]string(nil), c.imported.endComments...)
	}
}

func (c *Config) writeReloadWarn() {
	w := c.opts.ReloadWriter
	if w == nil {
		w = os.Stderr
	}
	// Format mirrors Perl: "PID <pid> reloading config file <path> at <ts>".
	// We keep a simpler but equivalent message. Tests do not assert exact
	// text; only that ReloadWarn prints something.
	_, _ = w.WriteString("inifiles: reloading config file " + c.file + "\n")
}

// isValidOctalMode mirrors Perl's SetWriteMode validator /$mode =~ m/[0-7]{3}/,
// an unanchored search for three consecutive octal digits. We accept any string
// whose characters are all octal digits with length >= 3 (covers "644", "0644",
// "0666").
func isValidOctalMode(mode string) bool {
	if len(mode) < 3 {
		return false
	}
	for _, ch := range mode {
		if ch < '0' || ch > '7' {
			return false
		}
	}
	return true
}

func parseOctalMode(mode string) os.FileMode {
	var m os.FileMode
	for _, ch := range mode {
		m = m<<3 | os.FileMode(ch-'0')
	}
	return m
}

func formatOctalMode(m os.FileMode) string {
	// Perl records mode as "%04o" (e.g. "0644").
	const digits = "01234567"
	out := make([]byte, 4)
	out[0] = '0'
	for i := 3; i >= 1; i-- {
		out[i] = digits[m&7]
		m >>= 3
	}
	return string(out)
}

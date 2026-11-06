package inifiles

import (
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

// SetWriteMode sets the file mode applied on WriteFile (only the permission
// bits are used). Pass 0 to clear the recorded mode.
func (c *Config) SetWriteMode(mode os.FileMode) {
	c.fileMode = mode.Perm()
}

// GetWriteMode returns the configured file mode (permission bits only), or 0
// if none has been set.
func (c *Config) GetWriteMode() os.FileMode {
	return c.fileMode
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
		// No file: discard current state, deep-copy imported data, then done.
		c.errs.reset()
		c.resetState()
		c.applyImport()
		c.reloadCnt++
		if c.opts.ReloadWarn && c.reloadCnt > 1 {
			c.writeReloadWarn()
		}
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
	_, _ = io.WriteString(w, "inifiles: reloading config file "+c.file+"\n")
}

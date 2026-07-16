package inifiles

import (
	"io"
	"os"
)

// WriteTo writes the configuration to w using the detected line ending (or
// "\n" when none). It returns nil on success.
func (c *Config) WriteTo(w io.Writer) (int64, error) {
	return c.writeTo(w, false)
}

// WriteToDelta writes only sections/parameters touched locally (plus deletion
// tombstones), producing a delta file suitable for re-applying atop an import.
func (c *Config) WriteToDelta(w io.Writer) (int64, error) {
	return c.writeTo(w, true)
}

// WriteFile writes the configuration to path atomically (temp file + rename)
// and applies the recorded file mode.
func (c *Config) WriteFile(path string) error {
	return c.writeFileTo(path, false)
}

// WriteFileDelta writes a delta file to path atomically.
func (c *Config) WriteFileDelta(path string) error {
	return c.writeFileTo(path, true)
}

// Rewrite rewrites the configured file (mirrors Perl RewriteConfig).
func (c *Config) Rewrite() error {
	if c.file == "" {
		return nil
	}
	return c.WriteFile(c.file)
}

func (c *Config) writeTo(w io.Writer, delta bool) (int64, error) {
	_ = delta
	return 0, errNotImplemented
}

func (c *Config) writeFileTo(path string, delta bool) error {
	_ = path
	_ = delta
	return errNotImplemented
}

// ensureDirStat records the file mode of path if not already set.
func (c *Config) ensureDirStat(path string) error {
	if c.fileMode != 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil // ignore; mode stays unset
	}
	c.fileMode = info.Mode().Perm()
	return nil
}
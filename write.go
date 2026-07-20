package inifiles

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// WriteTo writes the configuration to w using the detected line ending (or
// "\n" when none). It returns the number of bytes written.
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

func (c *Config) ors() string {
	if len(c.lineEnds) > 0 {
		return string(c.lineEnds)
	}
	return "\n"
}

func (c *Config) orRunes() []rune {
	if len(c.lineEnds) == 0 {
		return []rune{'\n'}
	}
	set := map[rune]bool{}
	for _, b := range c.lineEnds {
		set[rune(b)] = true
	}
	out := make([]rune, 0, len(set))
	// Preserve a stable order from lineEnds, then additions.
	for _, b := range c.lineEnds {
		r := rune(b)
		if set[r] {
			out = append(out, r)
			delete(set, r)
		}
	}
	for r := range set {
		out = append(out, r)
	}
	return out
}

func splitOnOrsRunes(s string, runes []rune) []string {
	if s == "" {
		return []string{""}
	}
	set := make(map[byte]struct{}, len(runes))
	for _, r := range runes {
		if r < 128 {
			set[byte(r)] = struct{}{}
		}
	}
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if _, exists := set[s[i]]; exists {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func (c *Config) outputLines(p *param) []string {
	if p.val.isMulti {
		return p.val.lines
	}
	return splitOnOrsRunes(p.val.scalar(), c.orRunes())
}

func (c *Config) ensureEndComment(endComment string) string {
	if endComment == "" {
		return ""
	}
	return " " + string(c.opts.CommentChar) + " " + endComment
}

func (c *Config) writeTo(w io.Writer, delta bool) (int64, error) {
	ors := c.ors()
	var b strings.Builder

	list := c.sections
	if delta {
		list = c.mySects
	}

	position := 0
	for _, s := range list {
		if delta && s.deleted {
			// v{$sect} undefined: emit a deletion marker in delta mode.
			b.WriteString(string(c.opts.CommentChar) + " [" + s.name + "] is deleted")
			b.WriteString(ors)
			position++
			continue
		}
		if !delta && s.deleted {
			// Should not appear in c.sections, but guard anyway.
			position++
			continue
		}
		if position > 0 {
			b.WriteString(ors)
		}
		for _, cm := range s.comments {
			b.WriteString(cm)
			b.WriteString(ors)
		}
		if !(c.fallbackUsed && s.name == c.fallback) {
			b.WriteString("[" + s.name + "]")
			b.WriteString(ors)
		}
		params := s.params
		if delta {
			params = s.myParms
		}
		for _, p := range params {
			if delta && p.deleted {
				b.WriteString(string(c.opts.CommentChar) + " " + p.name + " is deleted")
				b.WriteString(ors)
				continue
			}
			if p.deleted {
				continue
			}
			for _, cm := range p.comments {
				b.WriteString(cm)
				b.WriteString(ors)
			}
			if p.val.isNil() {
				continue
			}
			c.writeParam(&b, p, ors)
		}
		position++
	}

	// End-of-file comments.
	for _, cm := range c.endComments {
		b.WriteString(cm)
		b.WriteString(ors)
	}

	out := b.String()
	if runtime.GOOS == "windows" {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	n, err := io.WriteString(w, out)
	return int64(n), err
}

func (c *Config) writeParam(b *strings.Builder, p *param, ors string) {
	vals := c.outputLines(p)
	endComment := c.ensureEndComment(p.trailing)

	if len(vals) == 0 {
		b.WriteString(p.name + "=")
		b.WriteString(ors)
		return
	}
	single := len(vals) == 1 || c.opts.NoMultiline
	if single {
		for i := 0; i < len(vals)-1; i++ {
			b.WriteString(p.name + "=" + vals[i])
			b.WriteString(ors)
		}
		b.WriteString(p.name + "=" + vals[len(vals)-1] + endComment)
		b.WriteString(ors)
		return
	}
	eotmark := c.calcEOTMark(p, vals)
	b.WriteString(p.name + "= <<" + eotmark)
	b.WriteString(ors)
	for i := 0; i < len(vals)-1; i++ {
		b.WriteString(vals[i])
		b.WriteString(ors)
	}
	b.WriteString(vals[len(vals)-1] + endComment)
	b.WriteString(ors)
	b.WriteString(eotmark)
	b.WriteString(ors)
}

func (c *Config) calcEOTMark(p *param, vals []string) string {
	mark := "EOT"
	if p.hasEOT {
		mark = p.eot
	}
	joined := strings.Join(vals, " ")
	// Append uppercase letters until the marker no longer appears in the value.
	for strings.Contains(joined, mark) {
		mark += "A"
	}
	return mark
}

func (c *Config) writeFileTo(path string, delta bool) error {
	// Mirror Perl's _write_config_to_filename: if the destination exists but is
	// not writable, refuse to write. (The subsequent atomic temp+rename would
	// otherwise bypass a read-only destination's mode.)
	if _, err := os.Stat(path); err == nil {
		w, werr := os.OpenFile(path, os.O_WRONLY, 0)
		if werr != nil {
			return werr
		}
		_ = w.Close()
	}

	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	// Ensure the recorded file mode is set from an existing target.
	c.ensureDirStat(path)

	tmp, err := os.CreateTemp(dir, "temp.ini-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if _, err := c.writeTo(tmp, delta); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	if c.fileMode != 0 {
		_ = os.Chmod(path, c.fileMode)
	}
	return nil
}

func (c *Config) ensureDirStat(path string) error {
	if c.fileMode != 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	c.fileMode = info.Mode().Perm()
	return nil
}

var _ = errors.New

package inifiles

import "errors"

// ErrNotFound is returned when a section or parameter does not exist.
var ErrNotFound = errors.New("inifiles: section or parameter not found")

// Val returns the value of sect.parm joined with newlines if multi-valued. The
// boolean reports whether the parameter (or a default-section fallback) was
// found.
func (c *Config) Val(sect, parm string) (string, bool) {
	p, ok := c.lookupParamWithFallback(sect, parm)
	if !ok {
		return "", false
	}
	return p.val.scalar(), true
}

// ValWithDefault returns the value of sect.parm, or def when not found (mirrors
// Perl val's third argument default).
func (c *Config) ValWithDefault(sect, parm, def string) string {
	v, ok := c.Val(sect, parm)
	if !ok {
		return def
	}
	return v
}

// Vals returns the value of sect.parm as a slice (one element for scalar
// values, multiple for here-doc/accumulated). The boolean reports whether the
// parameter (or a default-section fallback) was found. A missing parameter
// returns nil, false (mirrors Perl list-context val returning an empty list).
func (c *Config) Vals(sect, parm string) ([]string, bool) {
	p, ok := c.lookupParamWithFallback(sect, parm)
	if !ok {
		return nil, false
	}
	return p.val.slice(), true
}

// lookupParamWithFallback finds the parameter in sect, falling back to the
// default section when missing (and sect != default). Returns nil,false when
// not found anywhere.
func (c *Config) lookupParamWithFallback(sect, parm string) (*param, bool) {
	if sect == "" || parm == "" {
		return nil, false
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	if s := c.sectionOf(sect); s != nil {
		if p := s.param(name); p != nil {
			return p, true
		}
	}
	if c.defaultSect != "" && c.caseify(sect) != c.defaultSect {
		if s := c.sectionOf(c.defaultSect); s != nil {
			if p := s.param(name); p != nil {
				return p, true
			}
		}
	}
	return nil, false
}

// Exists reports whether sect.parm is present, ignoring the default section.
func (c *Config) Exists(sect, parm string) bool {
	if sect == "" || parm == "" {
		return false
	}
	s := c.sectionOf(sect)
	if s == nil {
		return false
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	return s.param(name) != nil
}

func (c *Config) findParam(sect, parm string) *param {
	s := c.sectionOf(sect)
	if s == nil {
		return nil
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	return s.param(name)
}

// NewVal creates sect.parm with the given values, creating the section if
// needed. If multiple values are given the parameter is multi-valued (here-doc
// on output).
func (c *Config) NewVal(sect, parm string, vals ...string) error {
	if sect == "" || parm == "" {
		return nil
	}
	c.AddSection(sect)
	s := c.sectionOf(sect)
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	if s.param(name) == nil {
		s.addParam(&param{name: name})
	} else {
		// Replace existing (mirrors newval overwriting in-file/imported value).
		s.deleteParam(name)
		s.addParam(&param{name: name})
	}
	p := s.param(name)
	c.touchParam(s, p)
	if len(vals) > 1 {
		p.val = multiValue(vals)
		if !p.hasEOT {
			p.eot = "EOT"
			p.hasEOT = true
		}
	} else if len(vals) == 1 {
		p.val = singleValue(vals[0])
	} else {
		p.val = singleValue("")
	}
	return nil
}

// SetVal sets an existing parameter's value(s). Returns ErrNotFound if the
// parameter does not already exist (use NewVal to create).
func (c *Config) SetVal(sect, parm string, vals ...string) error {
	s := c.sectionOf(sect)
	if s == nil {
		return ErrNotFound
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	p := s.param(name)
	if p == nil {
		return ErrNotFound
	}
	c.touchParam(s, p)
	if len(vals) > 1 {
		p.val = multiValue(vals)
		p.eot = "EOT"
		p.hasEOT = true
	} else if len(vals) == 1 {
		p.val = singleValue(vals[0])
	} else {
		p.val = singleValue("")
	}
	return nil
}

// PushVal appends values to an existing multi-valued (or scalar -> array)
// parameter. Returns ErrNotFound if the parameter does not exist.
func (c *Config) PushVal(sect, parm string, vals ...string) error {
	s := c.sectionOf(sect)
	if s == nil {
		return ErrNotFound
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	p := s.param(name)
	if p == nil {
		return ErrNotFound
	}
	if len(vals) == 0 {
		return nil
	}
	c.touchParam(s, p)
	if !p.hasEOT {
		p.eot = "EOT"
		p.hasEOT = true
	}
	lines := p.val.lines
	if len(lines) == 0 {
		lines = []string{""}
	}
	p.val.lines = append(lines, vals...)
	p.val.isMulti = true
	return nil
}

// DelVal deletes sect.parm, recording a tombstone for delta output.
func (c *Config) DelVal(sect, parm string) error {
	if sect == "" || parm == "" {
		return nil
	}
	s := c.sectionOf(sect)
	if s == nil {
		return nil
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	if p := s.param(name); p != nil {
		s.deleteParam(name)
		p.deleted = true
		c.touchParam(s, p)
	}
	return nil
}

// stripPHPTailArray removes a trailing "[]" from name (PHPCompat).
func stripPHPTailArray(name string) string {
	if len(name) >= 2 && name[len(name)-2:] == "[]" {
		return name[:len(name)-2]
	}
	return name
}

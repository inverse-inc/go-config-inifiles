package inifiles

// SetSectionComment sets the comment lines appearing above sect, prefixing any
// line that lacks an allowed comment leader with "<commentChar> ". The section
// is created if it does not exist. Returns the count of comment lines. An
// empty sect returns ErrEmptyArg. Passing no comments is a no-op (0, nil).
func (c *Config) SetSectionComment(sect string, comments ...string) (int, error) {
	if sect == "" {
		return 0, ErrEmptyArg
	}
	if len(comments) == 0 {
		return 0, nil
	}
	c.AddSection(sect)
	s := c.sectionOf(sect)
	s.touched = true
	s.comments = c.markup(comments)
	return len(s.comments), nil
}

// GetSectionComment returns the section's comment lines (verbatim, prefixed as
// read). Returns nil,false if the section or its comment is absent.
func (c *Config) GetSectionComment(sect string) ([]string, bool) {
	s := c.sectionOf(sect)
	if s == nil || len(s.comments) == 0 {
		return nil, false
	}
	return append([]string(nil), s.comments...), true
}

// DeleteSectionComment removes the comment lines above sect. Deleting the
// comment on a missing section is a no-op (returns nil). An empty sect returns
// ErrEmptyArg.
func (c *Config) DeleteSectionComment(sect string) error {
	if sect == "" {
		return ErrEmptyArg
	}
	s := c.sectionOf(sect)
	if s == nil {
		return nil
	}
	s.touched = true
	s.comments = nil
	return nil
}

// SetParameterComment sets the comment lines appearing above sect.parm,
// prefixing any line lacking an allowed comment leader. Returns the count of
// comment lines. An empty sect or parm returns ErrEmptyArg. Passing no comments
// is a no-op (0, nil).
func (c *Config) SetParameterComment(sect, parm string, comments ...string) (int, error) {
	if sect == "" || parm == "" {
		return 0, ErrEmptyArg
	}
	if len(comments) == 0 {
		return 0, nil
	}
	p := c.findParam(sect, parm)
	if p == nil {
		// Perl allows setting a comment for a non-existing parameter; create
		// the param-less comment by ensuring the section/param exist? Perl
		// stores pCMT even if the param doesn't exist. We need a param entry to
		// hold the comment, so create it with an empty (nil) value marker.
		c.AddSection(sect)
		s := c.sectionOf(sect)
		name := c.caseify(parm)
		if c.opts.PHPCompat {
			name = stripPHPTailArray(name)
		}
		if s.param(name) == nil {
			s.addParam(&param{name: name, touched: true, val: singleValue("")})
		}
		p = s.param(name)
	}
	p.touched = true
	p.comments = c.markup(comments)
	return len(p.comments), nil
}

// GetParameterComment returns sect.parm's comment lines, or nil,false if absent.
func (c *Config) GetParameterComment(sect, parm string) ([]string, bool) {
	p := c.findParam(sect, parm)
	if p == nil || len(p.comments) == 0 {
		return nil, false
	}
	return append([]string(nil), p.comments...), true
}

// DeleteParameterComment removes sect.parm's comment lines. Deleting the
// comment on a missing parameter is a no-op (returns nil). An empty sect or
// parm returns ErrEmptyArg.
func (c *Config) DeleteParameterComment(sect, parm string) error {
	if sect == "" || parm == "" {
		return ErrEmptyArg
	}
	p := c.findParam(sect, parm)
	if p == nil {
		return nil
	}
	p.touched = true
	p.comments = nil
	return nil
}

// SetParameterTrailingComment sets the end-of-line comment for sect.parm. The
// parameter must exist; a missing parameter returns ErrNotFound. An empty sect
// or parm returns ErrEmptyArg. An empty cmt is a no-op (returns nil).
func (c *Config) SetParameterTrailingComment(sect, parm, cmt string) error {
	if sect == "" || parm == "" {
		return ErrEmptyArg
	}
	if cmt == "" {
		return nil
	}
	p := c.findParam(sect, parm)
	if p == nil {
		return ErrNotFound
	}
	p.touched = true
	p.trailing = cmt
	return nil
}

// GetParameterTrailingComment returns the end-of-line comment for sect.parm,
// or "" if none. The boolean reports whether the parameter exists.
func (c *Config) GetParameterTrailingComment(sect, parm string) (string, bool) {
	p := c.findParam(sect, parm)
	if p == nil {
		return "", false
	}
	return p.trailing, true
}

func (c *Config) markup(comments []string) []string {
	return markupComments(comments, c.opts.CommentChar, c.opts.AllowedCommentChars)
}

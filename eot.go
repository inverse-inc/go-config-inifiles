package inifiles

// SetParameterEOT sets the here-doc terminator marker for the parameter.
// Once set, the parameter serializes as a here-doc block on write. An empty
// sect, parm, or eot returns ErrEmptyArg. A missing parameter returns
// ErrNotFound.
func (c *Config) SetParameterEOT(sect, parm, eot string) error {
	if sect == "" || parm == "" || eot == "" {
		return ErrEmptyArg
	}
	p := c.findParam(sect, parm)
	if p == nil {
		return ErrNotFound
	}
	p.touched = true
	p.eot = eot
	p.hasEOT = true
	return nil
}

// GetParameterEOT returns the here-doc terminator marker for the parameter, or
// "" if none was set. A multi-valued param auto-created without an explicit
// marker returns "EOT" (the default), mirroring Perl.
func (c *Config) GetParameterEOT(sect, parm string) string {
	p := c.findParam(sect, parm)
	if p == nil {
		return ""
	}
	if p.hasEOT {
		return p.eot
	}
	if p.val.isMulti {
		return "EOT"
	}
	return ""
}

// DeleteParameterEOT removes the here-doc terminator marker from the parameter.
// A multi-valued param reverts to the default "EOT" marker on output. Deleting
// the marker on a missing parameter is a no-op (returns nil). An empty sect or
// parm returns ErrEmptyArg.
func (c *Config) DeleteParameterEOT(sect, parm string) error {
	if sect == "" || parm == "" {
		return ErrEmptyArg
	}
	p := c.findParam(sect, parm)
	if p == nil {
		return nil
	}
	p.touched = true
	p.eot = ""
	p.hasEOT = false
	return nil
}

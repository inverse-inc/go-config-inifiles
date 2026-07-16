package inifiles

// SetParameterEOT sets the here-doc terminator marker for the parameter.
// Once set, the parameter serializes as a here-doc block on write.
func (c *Config) SetParameterEOT(sect, parm, eot string) error {
	if sect == "" || parm == "" || eot == "" {
		return nil
	}
	p := c.findParam(sect, parm)
	if p == nil {
		return nil
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
// A multi-valued param reverts to the default "EOT" marker on output.
func (c *Config) DeleteParameterEOT(sect, parm string) error {
	p := c.findParam(sect, parm)
	if p == nil {
		return nil
	}
	p.touched = true
	p.eot = ""
	p.hasEOT = false
	return nil
}
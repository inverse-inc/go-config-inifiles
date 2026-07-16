package inifiles

// touchSection marks s as "ours" (mirrors _touch_section adding to mysects).
func (c *Config) touchSection(s *section) {
	if s.touched {
		return
	}
	s.touched = true
	c.mySects = append(c.mySects, s)
}

// touchParam marks p and its section as "ours" (mirrors _touch_parameter).
func (c *Config) touchParam(s *section, p *param) {
	c.touchSection(s)
	if p.touched {
		return
	}
	p.touched = true
	s.myParms = append(s.myParms, p)
}

// Sections returns the ordered list of section names (including imported ones).
func (c *Config) Sections() []string {
	out := make([]string, 0, len(c.sections))
	for _, s := range c.sections {
		out = append(out, s.name)
	}
	return out
}

// SectionExists reports whether a section named sect is present.
func (c *Config) SectionExists(sect string) bool {
	if sect == "" {
		return false
	}
	_, ok := c.sectIdx[c.caseify(sect)]
	return ok
}

func (c *Config) sectionOf(sect string) *section {
	if sect == "" {
		return nil
	}
	return c.sectIdx[c.caseify(sect)]
}

// AddSection creates a section if it does not already exist.
func (c *Config) AddSection(sect string) error {
	if sect == "" {
		return nil
	}
	name := c.caseify(sect)
	if _, ok := c.sectIdx[name]; ok {
		return nil
	}
	s := &section{name: name}
	c.addSectionInternal(s, true)
	return nil
}

func (c *Config) addSectionInternal(s *section, ourOwn bool) {
	c.sectIdx[s.name] = s
	c.sections = append(c.sections, s)
	if ourOwn {
		c.touchSection(s)
	}
	c.setGroupMember(s.name)
}

// DeleteSection removes sect and all its parameters, recording a tombstone for
// delta output.
func (c *Config) DeleteSection(sect string) error {
	name := c.caseify(sect)
	s, ok := c.sectIdx[name]
	if !ok {
		return nil
	}
	delete(c.sectIdx, name)
	for i, ss := range c.sections {
		if ss == s {
			c.sections = append(c.sections[:i], c.sections[i+1:]...)
			break
		}
	}
	s.deleted = true
	c.touchSection(s) // ensures it appears in delta as a deleted entry
	c.removeGroupMember(name)
	return nil
}

// CopySection copies old into newName (no overwrite). When includeGroup is
// true, group members of old are recursively copied to counterparts with the
// name prefix replaced.
func (c *Config) CopySection(old, newName string, includeGroup bool) error {
	if old == "" || newName == "" {
		return nil
	}
	src := c.sectionOf(old)
	if src == nil {
		return nil
	}
	newNameC := c.caseify(newName)
	if _, ok := c.sectIdx[newNameC]; ok {
		return nil
	}
	dst := deepCopySection(src)
	dst.name = newNameC
	dst.imported = false
	dst.touched = false
	c.addSectionInternal(dst, true)
	if includeGroup {
		for _, member := range c.GroupMembers(old) {
			repl := copySectionPrefix(member, old, newName)
			_ = c.CopySection(member, repl, true)
		}
	}
	return nil
}

// RenameSection copies old to new and then deletes old.
func (c *Config) RenameSection(old, newName string, includeGroup bool) error {
	if err := c.CopySection(old, newName, includeGroup); err != nil {
		return err
	}
	return c.DeleteSection(old)
}

// Parameters returns the ordered parameter names of sect (including imported).
func (c *Config) Parameters(sect string) []string {
	s := c.sectionOf(sect)
	if s == nil {
		return nil
	}
	return s.paramNames()
}

// Delete removes every section from the configuration.
func (c *Config) Delete() {
	c.sections = nil
	c.sectIdx = map[string]*section{}
	c.groups = map[string][]string{}
	c.groupOrder = nil
	c.endComments = nil
	c.mySects = nil
}

func copySectionPrefix(member, old, newName string) string {
	if len(member) >= len(old) && member[:len(old)] == old {
		return newName + member[len(old):]
	}
	return member
}
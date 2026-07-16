package inifiles

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

// sectionOf returns the internal *section for sect (caseified), or nil.
func (c *Config) sectionOf(sect string) *section {
	if sect == "" {
		return nil
	}
	return c.sectIdx[c.caseify(sect)]
}

// AddSection creates a section if it does not already exist. Returns nil if
// the section already existed (no-op success), and an error only if sect is
// empty.
func (c *Config) AddSection(sect string) error {
	if sect == "" {
		return nil
	}
	name := c.caseify(sect)
	if s, ok := c.sectIdx[name]; ok {
		_ = s
		return nil
	}
	s := &section{name: name}
	c.addSectionInternal(s, true)
	return nil
}

// addSectionInternal registers a section, marks it touched (if ourOwn) and
// enrolls it in any derived group.
func (c *Config) addSectionInternal(s *section, ourOwn bool) {
	c.sectIdx[s.name] = s
	c.sections = append(c.sections, s)
	if ourOwn {
		s.touched = true
	}
	c.setGroupMember(s.name)
}

// DeleteSection removes sect and all its parameters. Returns nil on success or
// if the section was absent.
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
	if s.touched {
		// Record as deleted for delta output but drop from ordering.
		c.recordDeletedSection(name)
	}
	c.removeGroupMember(name)
	return nil
}

// recordDeletedSection keeps a tombstone for delta writes.
func (c *Config) recordDeletedSection(name string) {
	c.deletedSections = append(c.deletedSections, deletedEntry{name: name})
}

// CopySection copies old into new (no overwrite: returns an error if new
// already exists). When includeGroup is true, group members of old are
// recursively copied to counterparts with the name prefix replaced.
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
		return nil // Perl: CopySection returns undef if new already exists
	}
	dst := deepCopySection(src)
	dst.name = newNameC
	dst.imported = false
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
	c.deletedSections = nil
}

// deletedEntry tracks section/param tombstones for delta writes.
type deletedEntry struct {
	name string
	kind byte // 0=section, 1=param
	sect string // for param deletes: owning section (caseified)
}

func (c *Config) recordDeletedParam(sect, name string) {
	c.deletedSections = append(c.deletedSections, deletedEntry{name: name, kind: 1, sect: sect})
}

func copySectionPrefix(member, old, newName string) string {
	if len(member) >= len(old) && member[:len(old)] == old {
		return newName + member[len(old):]
	}
	return member
}
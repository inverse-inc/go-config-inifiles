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

// AddSection creates a section if it does not already exist. It is idempotent:
// re-adding an existing section is a no-op (returns nil). An empty sect returns
// ErrEmptyArg.
func (c *Config) AddSection(sect string) error {
	if sect == "" {
		return ErrEmptyArg
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
// delta output. Deleting a missing section is a no-op (returns nil). An empty
// sect returns ErrEmptyArg.
func (c *Config) DeleteSection(sect string) error {
	if sect == "" {
		return ErrEmptyArg
	}
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
// name prefix replaced; existing counterparts are skipped and the rest of the
// group is still copied. An empty old or newName returns ErrEmptyArg. A missing
// old section returns ErrNotFound. An existing destination returns
// ErrAlreadyExists.
func (c *Config) CopySection(old, newName string, includeGroup bool) error {
	if old == "" || newName == "" {
		return ErrEmptyArg
	}
	src := c.sectionOf(old)
	if src == nil {
		return ErrNotFound
	}
	newNameC := c.caseify(newName)
	if _, ok := c.sectIdx[newNameC]; ok {
		return ErrAlreadyExists
	}
	return c.copySectionInternal(src, newNameC, old, newName, includeGroup)
}

// copySectionInternal performs the copy assuming src is non-nil and the
// destination does not exist. It recurses into group members, skipping any
// counterpart whose destination already exists (continue copying the rest).
func (c *Config) copySectionInternal(src *section, newNameC, old, newName string, includeGroup bool) error {
	dst := deepCopySection(src)
	dst.name = newNameC
	dst.imported = false
	dst.touched = false
	c.addSectionInternal(dst, true)
	if includeGroup {
		for _, member := range c.GroupMembers(old) {
			repl := copySectionPrefix(member, old, newName)
			// Skip counterparts whose destination already exists; continue
			// with the remaining group members.
			if c.sectionOf(repl) != nil {
				continue
			}
			if msrc := c.sectionOf(member); msrc == nil {
				continue
			} else {
				_ = c.copySectionInternal(msrc, c.caseify(repl), member, repl, true)
			}
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

// IsImported reports whether sect.parm originates from the imported config
// (i.e., the section is imported and the parameter has not been locally modified).
func (c *Config) IsImported(sect, parm string) bool {
	if sect == "" || parm == "" {
		return false
	}
	s := c.sectionOf(sect)
	if s == nil || !s.imported {
		return false
	}
	name := c.caseify(parm)
	if c.opts.PHPCompat {
		name = stripPHPTailArray(name)
	}
	p := s.param(name)
	if p == nil {
		return false
	}
	return !p.touched
}

// MyParameters returns the ordered parameter names of sect that are either in a
// non-imported section or have been modified (touched).
func (c *Config) MyParameters(sect string) []string {
	s := c.sectionOf(sect)
	if s == nil {
		return nil
	}
	if !s.imported {
		return s.paramNames()
	}
	var out []string
	for _, p := range s.params {
		if p.touched {
			out = append(out, p.name)
		}
	}
	return out
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

// ResortSections reorders the given sections into the supplied order, keeping
// every section not listed at its relative position around the reordered block
// (mirrors Perl Config::IniFiles::ResortSections). Every name in sects must
// refer to an existing section; otherwise ResortSections returns ErrNotFound
// and leaves the order untouched. An empty sects list (or a single-element
// list) is a no-op.
func (c *Config) ResortSections(sects []string) error {
	if len(sects) == 0 || len(sects) == 1 {
		return nil
	}
	// Normalize to the storage form (caseified) and verify all exist.
	names := make([]string, len(sects))
	for i, s := range sects {
		names[i] = c.caseify(s)
		if _, ok := c.sectIdx[names[i]]; !ok {
			return ErrNotFound
		}
	}

	// Full reorder: the supplied list covers every existing section.
	if len(c.sections) == len(names) {
		c.rebuildSectionsOrder(names)
		return nil
	}

	// Partial reorder: position the block where its first element currently is,
	// keeping unlisted sections in their original order around it.
	firstIndex := -1
	for i, s := range c.sections {
		if s.name == names[0] {
			firstIndex = i
			break
		}
	}
	if firstIndex < 0 {
		return ErrNotFound
	}

	temp := make(map[string]struct{}, len(names))
	for _, n := range names {
		temp[n] = struct{}{}
	}

	var newOrder []*section
	for i := 0; i <= firstIndex; i++ {
		if _, in := temp[c.sections[i].name]; !in {
			newOrder = append(newOrder, c.sections[i])
		}
	}
	for _, n := range names {
		newOrder = append(newOrder, c.sectIdx[n])
	}
	for i := firstIndex + 1; i < len(c.sections); i++ {
		if _, in := temp[c.sections[i].name]; !in {
			newOrder = append(newOrder, c.sections[i])
		}
	}
	c.sections = newOrder
	return nil
}

// rebuildSectionsOrder re-seats c.sections to follow order, which must contain
// exactly the currently-known section names (no duplicates, none missing).
func (c *Config) rebuildSectionsOrder(order []string) {
	out := make([]*section, 0, len(order))
	for _, n := range order {
		out = append(out, c.sectIdx[n])
	}
	c.sections = out
}

func copySectionPrefix(member, old, newName string) string {
	if len(member) >= len(old) && member[:len(old)] == old {
		return newName + member[len(old):]
	}
	return member
}

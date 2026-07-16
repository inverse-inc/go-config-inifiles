package inifiles

import "strings"

// Groups returns the names of all groups derived from section names.
func (c *Config) Groups() []string {
	out := make([]string, len(c.groupOrder))
	copy(out, c.groupOrder)
	return out
}

// GroupMembers returns the section names belonging to group, or nil if group is
// unknown.
func (c *Config) GroupMembers(group string) []string {
	if group == "" {
		return nil
	}
	g := c.caseify(group)
	if c.opts.NoCase {
		// Group keys are stored under the original-case first word as parsed
		// from the section name (which, under nocase, is already lowercased).
		g = toLower(group)
	}
	members, ok := c.groups[g]
	if !ok {
		return nil
	}
	return append([]string(nil), members...)
}

// SetGroupMember enrolls sect in the group derived from its name (the first
// whitespace-delimited word, when the name contains a second word). This is
// invoked automatically by AddSection.
func (c *Config) SetGroupMember(sect string) {
	c.setGroupMember(sect)
}

func (c *Config) setGroupMember(sect string) {
	group, ok := groupFromName(sect)
	if !ok {
		return
	}
	key := group
	if c.opts.NoCase {
		key = toLower(group)
	}
	if _, exists := c.groups[key]; !exists {
		c.groups[key] = nil
		c.groupOrder = append(c.groupOrder, key)
	}
	for _, m := range c.groups[key] {
		if m == sect {
			return
		}
	}
	c.groups[key] = append(c.groups[key], sect)
}

// RemoveGroupMember removes sect from its derived group (if any).
func (c *Config) RemoveGroupMember(sect string) {
	c.removeGroupMember(sect)
}

func (c *Config) removeGroupMember(sect string) {
	group, ok := groupFromName(sect)
	if !ok {
		return
	}
	key := group
	if c.opts.NoCase {
		key = toLower(group)
	}
	members := c.groups[key]
	for i, m := range members {
		if m == sect {
			c.groups[key] = append(members[:i], members[i+1:]...)
			if len(c.groups[key]) == 0 {
				delete(c.groups, key)
				for j, g := range c.groupOrder {
					if g == key {
						c.groupOrder = append(c.groupOrder[:j], c.groupOrder[j+1:]...)
						break
					}
				}
			}
			return
		}
	}
}

// groupFromName derives the group name from a section name: the leading word
// before the first whitespace, when the name has a second word. Mirrors Perl
// /\A(\S+)\s+\S/.
func groupFromName(name string) (string, bool) {
	i := strings.IndexAny(name, " \t")
	if i <= 0 {
		return "", false
	}
	rest := name[i+1:]
	if strings.TrimSpace(rest) == "" {
		return "", false
	}
	return name[:i], true
}

package inifiles

// value is a tagged union: single-line scalar or multi-line here-doc/accumulated.
type value struct {
	lines   []string // len==1 for scalar, >1 for multi-valued
	isMulti bool     // true if this should serialize as here-doc (or repeated lines w/ nomultiline)
}

func singleValue(s string) value { return value{lines: []string{s}} }

func multiValue(lines []string) value { return value{lines: append([]string(nil), lines...), isMulti: true} }

func (v value) scalar() string {
	if len(v.lines) == 0 {
		return ""
	}
	if len(v.lines) == 1 {
		return v.lines[0]
	}
	// Join multi-line value with newline (Perl uses $/, default \n).
	out := ""
	for i, l := range v.lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func (v value) slice() []string {
	if len(v.lines) == 0 {
		return nil
	}
	return append([]string(nil), v.lines...)
}

func (v value) isNil() bool { return v.lines == nil }
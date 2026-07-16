package inifiles

import (
	"bytes"
	"regexp"
)

var (
	sectionRe = regexp.MustCompile(`^\s*\[\s*(\S|\S.*\S)\s*\]\s*$`)
	paramRe   = regexp.MustCompile(`^\s*([^=]*?[^=\s])\s*=\s*(.*)$`)
)

// parse parses data into c, mirroring Perl's ReadConfig parse loop.
func (c *Config) parse(data []byte) error {
	if err := validateCommentChars(c.opts.CommentChar, c.opts.AllowedCommentChars); err != nil {
		return err
	}

	lines, lineEnds := splitLines(data)
	c.lineEnds = lineEnds

	// Allowed comment runes: allowed + commentChar.
	allowed := map[rune]bool{c.opts.CommentChar: true}
	for _, r := range c.opts.AllowedCommentChars {
		allowed[r] = true
	}

	ps := &parserState{lines: lines, allowed: allowed}
	// First-line UTF-8 BOM strip handled in splitLines already.
	if !c.parseLoop(ps) {
		return c.errs.asError()
	}

	// Empty file (no sections at all) is an error unless AllowEmpty.
	if len(c.sections) == 0 && !c.opts.AllowEmpty {
		c.errs.add(0, "Empty file treated as error")
		return c.errs.asError()
	}

	// Ensure the default section exists (mirrors AddSection after parse).
	if c.opts.Default != "" {
		c.AddSection(c.opts.Default)
	}

	// Trailing comments become end-of-file comments.
	c.endComments = ps.pending

	if c.errs.has() {
		return c.errs.asError()
	}
	return nil
}

type parserState struct {
	lines   []string
	idx     int
	allowed map[rune]bool

	currSect string
	currParm string
	currVal  string
	currEnd  string
	pending  []string // accumulated comment lines awaiting attachment
}

func (ps *parserState) nextLine() (string, bool) {
	if ps.idx >= len(ps.lines) {
		return "", false
	}
	l := ps.lines[ps.idx]
	ps.idx++
	return l, true
}

// parseLoop returns false if a fatal break occurred (here-doc missing terminator
// or parameter outside section), true otherwise. Malformed lines are recorded
// as errors but parsing continues.
func (c *Config) parseLoop(ps *parserState) bool {
	for {
		line, ok := ps.nextLine()
		if !ok {
			return true
		}
		// 1. blank
		if isBlank(line) {
			continue
		}
		// 2. comment
		if lineStartsWithComment(line, ps.allowed) {
			c.handleComment(ps, line)
			continue
		}
		// 3. section
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			c.newSection(ps, m[1])
			continue
		}
		// 4. parameter
		if m := paramRe.FindStringSubmatch(line); m != nil {
			if !c.paramAssignment(ps, m[1], m[2]) {
				return false
			}
			continue
		}
		// 5. malformed
		c.errs.addf(ps.idx, "Line %d in file %s is malformed:\n\t%s", ps.idx, c.file, line)
		// continue parsing
	}
}

func isBlank(line string) bool {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ', '\t', '\r':
			continue
		default:
			return false
		}
	}
	return true
}

func (c *Config) handleComment(ps *parserState, line string) {
	// Negative deltas: "<commentChar> [.name.] is deleted"
	if c.opts.NegativeDeltas {
		prefix := string(c.opts.CommentChar) + " "
		suffix := " is deleted"
		if len(line) >= len(prefix)+len(suffix) && line[:len(prefix)] == prefix && line[len(line)-len(suffix):] == suffix {
			toDelete := line[len(prefix) : len(line)-len(suffix)]
			if len(toDelete) >= 2 && toDelete[0] == '[' && toDelete[len(toDelete)-1] == ']' {
				c.DeleteSection(toDelete[1 : len(toDelete)-1])
			} else {
				c.DelVal(ps.currSect, toDelete)
			}
			return
		}
	}
	// verbatim pending comment
	ps.pending = append(ps.pending, line)
}

func (c *Config) newSection(ps *parserState, name string) {
	name = c.caseify(name)
	ps.currSect = name
	c.AddSection(name)
	if s := c.sectionOf(name); s != nil {
		// Attach pending comments (verbatim, they already have leaders).
		if len(ps.pending) > 0 {
			s.comments = append([]string(nil), ps.pending...)
		}
	}
	ps.pending = nil
}

// paramAssignment returns false on fatal break (outside section / missing
// here-doc terminator).
func (c *Config) paramAssignment(ps *parserState, parm, value string) bool {
	parm = c.caseify(parm)
	if c.opts.PHPCompat {
		parm = stripPHPTailArray(parm)
		value = unwrapPHPQuotes(value)
	}
	ps.currVal = value
	ps.currEnd = ""

	// Fallback / no-section test.
	if ps.currSect == "" {
		if c.fallback != "" {
			ps.currSect = c.fallback
			c.fallbackUsed = true
			// Ensure the fallback section exists (mirrors newval's AddSection).
			c.AddSection(ps.currSect)
		} else {
			c.errs.addf(ps.idx, "%d: %s", ps.idx, "parameter found outside a section")
			return false
		}
	}
	ps.currParm = parm

	var vals []string
	var eotmark string
	hasEOT := false

	if rest, ok := hereDocMarker(value); ok {
		eotmark = rest
		hasEOT = true
		ok := c.readHereDoc(ps, eotmark, &vals)
		if !ok {
			c.errs.addf(ps.idx, `%d: %s`, ps.idx, `no end marker ("`+eotmark+`") found`)
			return false
		}
	} else {
		c.processContinue(ps, &ps.currVal)
		if c.opts.HandleTrailingComment {
			v, ec := splitTrailingComment(ps.currVal, ps.allowed)
			ps.currVal = v
			ps.currEnd = ec
		}
		vals = []string{ps.currVal}
	}

	c.loadValue(ps, vals, eotmark, hasEOT)
	return true
}

func hereDocMarker(value string) (string, bool) {
	if len(value) >= 2 && value[0] == '<' && value[1] == '<' {
		return value[2:], true
	}
	return "", false
}

func (c *Config) readHereDoc(ps *parserState, marker string, vals *[]string) bool {
	for {
		line, ok := ps.nextLine()
		if !ok {
			return false
		}
		if line == marker {
			return true
		}
		*vals = append(*vals, line)
	}
}

// processContinue mirrors _process_continue_val: while the value ends with `\`,
// strip it and append the next raw line.
func (c *Config) processContinue(ps *parserState, val *string) {
	if !c.opts.AllowContinue {
		return
	}
	for len(*val) > 0 && (*val)[len(*val)-1] == '\\' {
		*val = (*val)[:len(*val)-1]
		line, ok := ps.nextLine()
		if !ok {
			return
		}
		*val += line
	}
}

// splitTrailingComment splits value into (value, trailingComment) at the first
// allowed comment char, mirroring /(.*?)\s*[$allCmt]\s*(.*)$/.
func splitTrailingComment(val string, allowed map[rune]bool) (string, string) {
	// Find first index of an allowed comment rune (after a non-empty value).
	for i := 0; i < len(val); i++ {
		r := rune(val[i])
		if allowed[r] {
			left := val[:i]
			right := val[i+1:]
			// strip trailing whitespace of left
			left = trimTrailingWhitespace(left)
			// strip leading whitespace of right
			right = trimLeadingWhitespace(right)
			return left, right
		}
	}
	return val, ""
}

func trimTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
		i--
	}
	return s[:i]
}
func trimLeadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

// loadValue pushes or replaces per the Perl _ReadConfig_load_value rule, then
// attaches comments, EOT, and trailing comment.
func (c *Config) loadValue(ps *parserState, vals []string, eotmark string, hasEOT bool) {
	s := c.sectionOf(ps.currSect)
	if s == nil {
		return
	}
	p := s.param(ps.currParm)
	shouldPush := p != nil && s.touched && p.touched
	if shouldPush {
		c.pushRaw(s, p, vals)
	} else {
		c.newvalRaw(s, ps.currParm, vals)
		p = s.param(ps.currParm)
	}
	if p == nil {
		return
	}
	// Attach pending comments (verbatim).
	if len(ps.pending) > 0 {
		p.comments = append([]string(nil), ps.pending...)
	}
	ps.pending = nil
	// EOT marker.
	if hasEOT {
		p.eot = eotmark
		p.hasEOT = true
	}
	// Trailing comment (empty when feature off).
	p.trailing = ps.currEnd
}

// newvalRaw creates or overwrites a parameter within section s.
func (c *Config) newvalRaw(s *section, parm string, vals []string) {
	if s.param(parm) != nil {
		s.deleteParam(parm)
	}
	p := &param{name: parm, val: singleValue("")}
	if len(vals) > 1 {
		p.val = multiValue(vals)
		p.eot = "EOT"
		p.hasEOT = true
	} else if len(vals) == 1 {
		p.val = singleValue(vals[0])
	} else {
		p.val = singleValue("")
	}
	s.addParam(p)
	c.touchParam(s, p)
}

// pushRaw appends values to an existing parameter, arrayifying a scalar first.
func (c *Config) pushRaw(s *section, p *param, vals []string) {
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
}

// unwrapPHPQuotes unwraps surrounding matching quotes and unescapes doubled
// quotes (PHPCompat), mirroring Perl's _caseify php branch for values.
func unwrapPHPQuotes(v string) string {
	if len(v) < 2 {
		return v
	}
	q := v[0]
	if q != '"' && q != '\'' {
		return v
	}
	if v[len(v)-1] != q {
		return v
	}
	inner := v[1 : len(v)-1]
	// Collapse doubled inner quotes.
	doubled := []byte{q, q}
	inner = replaceAll(inner, string(doubled), "")
	if q == '"' {
		inner = replaceAll(inner, `\`+string(q), string(q))
	}
	return inner
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := ""
	for {
		i := indexBytes(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexBytes(s, sub string) int {
	return bytes.Index([]byte(s), []byte(sub))
}
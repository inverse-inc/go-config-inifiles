package inifiles

import "strings"

// param holds a single ordered parameter within a section.
type param struct {
	name     string
	val      value
	comments []string // lines above the parameter (verbatim, prefixed-as-read)
	trailing string   // end-of-line comment (HandleTrailingComment only)
	eot      string   // here-doc terminator marker; "" means unset -> defaults on output
	hasEOT   bool     // whether an EOT was explicitly set; distinguishes "" marker from absent
	touched  bool     // "ours" (in myparms)
	deleted  bool     // delta deletion marker (kept in myParms as a tombstone)
}

// section holds an ordered section.
type section struct {
	name     string
	params   []*param // live, ordered (all params incl imported; for normal output)
	myParms  []*param // touched params, ordered, retaining deleted tombstones (for delta)
	comments []string // lines above the section header (verbatim)
	imported bool     // came via Import
	touched  bool     // "ours" (in mysects)
	deleted  bool     // delta deletion marker (kept in mySects as a tombstone)
}

func (s *section) param(name string) *param {
	for _, p := range s.params {
		if p.name == name {
			return p
		}
	}
	return nil
}

func (s *section) paramIndex(name string) int {
	for i, p := range s.params {
		if p.name == name {
			return i
		}
	}
	return -1
}

func (s *section) addParam(p *param) {
	s.params = append(s.params, p)
}

func (s *section) deleteParam(name string) {
	i := s.paramIndex(name)
	if i >= 0 {
		s.params = append(s.params[:i], s.params[i+1:]...)
	}
}

func (s *section) paramNames() []string {
	out := make([]string, 0, len(s.params))
	for _, p := range s.params {
		out = append(out, p.name)
	}
	return out
}

func deepCopyComments(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func deepCopyParam(p *param) *param {
	return &param{
		name:     p.name,
		val:      value{lines: append([]string(nil), p.val.lines...), isMulti: p.val.isMulti},
		comments: deepCopyComments(p.comments),
		trailing: p.trailing,
		eot:      p.eot,
		hasEOT:   p.hasEOT,
		touched:  false,
	}
}

func deepCopySection(s *section) *section {
	out := &section{
		name:     s.name,
		comments: deepCopyComments(s.comments),
		imported: true,
	}
	for _, p := range s.params {
		out.params = append(out.params, deepCopyParam(p))
	}
	return out
}

// markupComments returns a copy of comments where each line lacking an allowed
// comment leader is prefixed with "<commentChar> ". Lines already starting with
// an allowed comment char (after optional whitespace) are kept verbatim.
func markupComments(comments []string, commentChar rune, allowedRunes []rune) []string {
	allowed := make(map[rune]bool)
	allowed[commentChar] = true
	for _, r := range allowedRunes {
		allowed[r] = true
	}
	prefix := string(commentChar) + " "
	out := make([]string, 0, len(comments))
	for _, line := range comments {
		if lineStartsWithComment(line, allowed) {
			out = append(out, line)
			continue
		}
		out = append(out, prefix+line)
	}
	return out
}

// lineStartsWithComment reports whether line, after optional leading
// whitespace, begins with one of the allowed comment runes.
func lineStartsWithComment(line string, allowed map[rune]bool) bool {
	i := 0
	for i < len(line) {
		r := rune(line[i])
		if r != ' ' && r != '\t' {
			return allowed[r]
		}
		i++
	}
	return false
}

// joinCommentLines joins comment lines with newline (mirrors Perl _return_comment
// scalar context joining with $/, default \n).
func joinCommentLines(lines []string) string {
	return strings.Join(lines, "\n")
}
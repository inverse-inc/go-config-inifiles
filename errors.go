package inifiles

import "fmt"

// ParseError describes an error encountered while reading a config.
type ParseError struct {
	Line    int
	Message string
	Context string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%d: %s", e.Line, e.Message)
	}
	return e.Message
}

func (e *ParseError) Unwrap() error { return nil }

type errorList struct {
	errs []ParseError
}

func (l *errorList) add(line int, msg string) {
	l.errs = append(l.errs, ParseError{Line: line, Message: msg})
}

func (l *errorList) addf(line int, format string, args ...any) {
	l.errs = append(l.errs, ParseError{Line: line, Message: fmt.Sprintf(format, args...)})
}

func (l *errorList) reset() { l.errs = nil }

func (l *errorList) has() bool { return len(l.errs) > 0 }

func (l *errorList) asError() error {
	if !l.has() {
		return nil
	}
	// Return a joined representation; callers can use Errors() for details.
	return &multiError{errs: append([]ParseError(nil), l.errs...)}
}

type multiError struct {
	errs []ParseError
}

func (m *multiError) Error() string {
	if len(m.errs) == 0 {
		return "no errors"
	}
	s := m.errs[0].Error()
	for _, e := range m.errs[1:] {
		s += "; " + e.Error()
	}
	return s
}

func (m *multiError) Errors() []ParseError { return m.errs }

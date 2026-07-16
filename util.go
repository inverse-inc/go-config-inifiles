package inifiles

import "strings"

// toLower lowercases a string. Mirrors Perl lc(), which is Unicode-aware; we
// delegate to strings.ToLower for the same coverage.
func toLower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s)
}

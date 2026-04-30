package data

import (
	"regexp"
	"strings"
)

var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string { return stripANSI(s) }

// SanitizeResourceName strips ANSI escapes and C0/DEL control characters from s,
// then truncates to 253 characters (Kubernetes DNS-1123 max length). Prevents
// terminal-control-sequence injection when displaying resource names in dialogs.
func SanitizeResourceName(s string) string {
	s = stripANSI(s)
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return -1
		}
		return r
	}, s)
	if len(s) > 253 {
		s = s[:253]
	}
	return s
}

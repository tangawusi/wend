package normalize

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func CleanText(s string) string {
	if s == "" {
		return ""
	}
	s = norm.NFC.String(s)

	var b strings.Builder
	b.Grow(len(s))

	var prevSpace bool
	var newlines int
	for _, r := range s {
		switch {
		case r == '\uFEFF' || r == '\u200B' || r == '\u200C' || r == '\u200D':
			continue
		case r == '\u00A0' || r == '\u2007' || r == '\u202F':
			r = ' '
		case unicode.Is(unicode.Cc, r) && r != '\n' && r != '\t':
			continue
		case unicode.Is(unicode.Cf, r):
			continue
		}
		switch r {
		case '\r':
			continue
		case '\n':
			newlines++
			if newlines > 2 {
				continue
			}
			prevSpace = false
			b.WriteRune('\n')
		case ' ', '\t':
			if prevSpace {
				continue
			}
			prevSpace = true
			b.WriteRune(' ')
		default:
			prevSpace = false
			newlines = 0
			b.WriteRune(r)
		}
	}

	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func ContentHash(title, body string) string {
	return hashString(title + "\x1f" + body)
}

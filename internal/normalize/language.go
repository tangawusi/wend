package normalize

import (
	"strings"
	"unicode"
)

func DetectLanguage(s string) string {
	if s == "" {
		return "und"
	}
	counts := map[string]int{}
	total := 0
	for _, r := range s {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			counts["zh"]++
		case r >= 0x3040 && r <= 0x309F:
			counts["ja"]++
		case r >= 0x30A0 && r <= 0x30FF:
			counts["ja"]++
		case r >= 0xAC00 && r <= 0xD7AF:
			counts["ko"]++
		case r >= 0x0400 && r <= 0x04FF:
			counts["ru"]++
		case r >= 0x0600 && r <= 0x06FF:
			counts["ar"]++
		case r >= 0x0590 && r <= 0x05FF:
			counts["he"]++
		case r >= 0x0900 && r <= 0x097F:
			counts["hi"]++
		case r >= 0x0E00 && r <= 0x0E7F:
			counts["th"]++
		default:
			if unicode.IsLetter(r) {
				counts["latin"]++
			}
		}
		total++
	}
	if total == 0 {
		return "und"
	}
	best, bestN := "", 0
	for k, v := range counts {
		if k == "latin" {
			continue
		}
		if v > bestN {
			best, bestN = k, v
		}
	}
	if bestN*100/total >= 20 {
		return best
	}
	return detectLatin(s)
}

func detectLatin(s string) string {
	lower := ""
	for _, r := range s {
		lower += string(unicode.ToLower(r))
	}
	scores := map[string]int{
		"en": countAny(lower, " the ", " and ", " of ", " to ", " in "),
		"es": countAny(lower, " el ", " la ", " los ", " las ", " de ", " que "),
		"fr": countAny(lower, " le ", " la ", " les ", " des ", " du ", " et "),
		"pt": countAny(lower, " o ", " a ", " os ", " as ", " de ", " que "),
		"de": countAny(lower, " der ", " die ", " das ", " und ", " ist ", " den "),
		"it": countAny(lower, " il ", " la ", " gli ", " di ", " che ", " e "),
	}
	best, bestN := "und", 0
	for k, v := range scores {
		if v > bestN {
			best, bestN = k, v
		}
	}
	return best
}

func countAny(haystack string, needles ...string) int {
	n := 0
	for _, needle := range needles {
		n += strings.Count(haystack, needle)
	}
	return n
}

package normalize

import "strings"

var byPrefixes = []string{"by ", "By ", "BY ", "written by ", "Written by "}

var authorNoise = map[string]bool{
	"admin": true, "administrator": true, "editor": true,
	"staff": true, "staff writer": true, "newsroom": true,
	"editorial": true, "editorial team": true, "news": true,
	"unknown": true, "n/a": true, "na": true, "none": true,
	"reuters": true, "associated press": true, "ap": true,
	"afp": true, "agence france-presse": true,
}

func CleanAuthor(raw string) string {
	a := CleanText(raw)
	if a == "" {
		return ""
	}
	if i := strings.LastIndex(a, "("); i >= 0 && strings.HasSuffix(a, ")") {
		if inner := strings.TrimSpace(a[i+1 : len(a)-1]); inner != "" {
			a = inner
		}
	}
	for _, p := range byPrefixes {
		a = strings.TrimPrefix(a, p)
	}
	a = strings.TrimSpace(a)
	if a == "" {
		return ""
	}
	if authorNoise[strings.ToLower(a)] {
		return ""
	}
	if len([]rune(a)) > 120 {
		return ""
	}
	return a
}

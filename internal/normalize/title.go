package normalize

import "strings"

func CleanTitle(raw, sourceName, sourceDomain string) string {
	t := CleanText(raw)
	if t == "" {
		return ""
	}
	for _, sep := range []string{" - ", " | ", " – ", " — ", " :: "} {
		idx := strings.LastIndex(t, sep)
		if idx < 0 {
			continue
		}
		suffix := strings.TrimSpace(t[idx+len(sep):])
		if isSiteSuffix(suffix, sourceName, sourceDomain) {
			t = strings.TrimSpace(t[:idx])
			break
		}
	}
	return t
}

func isSiteSuffix(suffix, sourceName, sourceDomain string) bool {
	if suffix == "" {
		return false
	}
	s := strings.ToLower(suffix)
	if sourceName != "" && s == strings.ToLower(sourceName) {
		return true
	}
	if sourceDomain != "" {
		root := strings.SplitN(sourceDomain, ".", 2)[0]
		if root != "" && (s == sourceDomain || strings.HasPrefix(s, root+" ") ||
			strings.Contains(s, " "+root+" ") || strings.HasSuffix(s, " "+root)) {
			return true
		}
	}
	switch s {
	case "news", "world news", "breaking news", "latest news",
		"home", "homepage", "the guardian", "reuters", "ap news",
		"al jazeera", "the new york times", "the washington post":
		return true
	}
	return false
}

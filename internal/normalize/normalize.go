// Package normalize turns raw crawled bytes into the canonical fields
// the rest of the system depends on.
//
// Everything here is pure: no I/O, no clock, no network. Given the same
// input, Article returns the same output, byte for byte.
package normalize

import (
	"strings"
	"time"
)

const Version = "normalize/v0.1.0"

const MinBodyRunes = 200

type Options struct {
	SourceDomain string
	SourceName   string
}

type Input struct {
	URL          string
	Title        string
	RawHTML      []byte
	FallbackBody string
	Author       string
	PublishedAt  *time.Time
	Language     string
}

type Output struct {
	URL          string
	CanonicalURL string
	Title        string
	Body         string
	Author       string
	PublishedAt  *time.Time
	Language     string
	ContentHash  string
}

func Article(in Input, opts Options) Output {
	canonical := CanonicalizeURL(in.URL)
	title := CleanTitle(in.Title, opts.SourceName, opts.SourceDomain)

	body := CleanText(ExtractBody(in.RawHTML))
	if len([]rune(body)) < MinBodyRunes {
		if cleaned := CleanText(in.FallbackBody); len([]rune(cleaned)) > len([]rune(body)) {
			body = cleaned
		}
	}

	author := CleanAuthor(in.Author)
	published := NormalizeTime(in.PublishedAt)

	lang := in.Language
	if lang == "" {
		lang = DetectLanguage(title + " " + body)
	}

	return Output{
		URL:          in.URL,
		CanonicalURL: canonical,
		Title:        title,
		Body:         body,
		Author:       author,
		PublishedAt:  published,
		Language:     lang,
		ContentHash:  ContentHash(title, body),
	}
}

func joinBlocks(blocks []string) string {
	cleaned := blocks[:0]
	for _, b := range blocks {
		if s := CleanText(b); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	return strings.Join(cleaned, "\n\n")
}

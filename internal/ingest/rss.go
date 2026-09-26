package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"wend.press/internal/domain"
)

type RSSAdapter struct {
	source domain.Source
	cfg    CollectorConfig
	http   *http.Client
}

func NewRSSAdapter(src domain.Source, cfg CollectorConfig) *RSSAdapter {
	return &RSSAdapter{
		source: src,
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.Timeout},
	}
}

func (a *RSSAdapter) Source() domain.Source { return a.source }

func (a *RSSAdapter) Discover(ctx context.Context) ([]Discovered, error) {
	if a.source.RSSURL == "" {
		return nil, fmt.Errorf("source %s has no rss_url", a.source.Name)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.source.RSSURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.cfg.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9, */*;q=0.5")

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rss: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(a.cfg.MaxBodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("read rss: %w", err)
	}
	return parseRSS(body, a.source)
}

type rssDoc struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
	Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
}

func parseRSS(body []byte, src domain.Source) ([]Discovered, error) {
	var doc rssDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}

	out := make([]Discovered, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		link := strings.TrimSpace(it.Link)
		if link == "" {
			link = strings.TrimSpace(it.GUID)
		}
		if link == "" {
			continue
		}
		author := strings.TrimSpace(it.Author)
		if author == "" {
			author = strings.TrimSpace(it.Creator)
		}
		out = append(out, Discovered{
			URL:          link,
			CanonicalURL: canonicalize(link),
			Title:        strings.TrimSpace(it.Title),
			Summary:      stripHTML(strings.TrimSpace(it.Description)),
			Author:       author,
			PublishedAt:  parseRSSDate(it.PubDate),
		})
	}
	return out, nil
}

func (a *RSSAdapter) Extract(ctx context.Context, url string) (*Extracted, error) {
	c, err := NewCollector(a.cfg)
	if err != nil {
		return nil, err
	}

	var raw []byte
	var paragraphs []string

	c.OnResponse(func(r *colly.Response) {
		raw = r.Body
	})
	c.OnHTML("article p, main p, .story-body p, .article-body p", func(e *colly.HTMLElement) {
		t := strings.TrimSpace(e.Text)
		if len(t) >= 40 {
			paragraphs = append(paragraphs, t)
		}
	})

	if err := c.Visit(url); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return &Extracted{
		RawHTML: raw,
		Body:    strings.Join(paragraphs, "\n\n"),
	}, nil
}

func canonicalize(raw string) string {
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		return raw[:i]
	}
	return raw
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

var rssDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
}

func parseRSSDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, f := range rssDateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

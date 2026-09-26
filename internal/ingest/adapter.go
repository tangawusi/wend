package ingest

import (
	"context"
	"time"

	"wend.press/internal/domain"
)

type Discovered struct {
	URL          string
	CanonicalURL string
	Title        string
	Summary      string
	Author       string
	PublishedAt  *time.Time
}

type Extracted struct {
	RawHTML []byte
	Body    string
	Author  string
}

type SourceAdapter interface {
	Source() domain.Source
	Discover(ctx context.Context) ([]Discovered, error)
	Extract(ctx context.Context, url string) (*Extracted, error)
}

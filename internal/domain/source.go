package domain

import "time"

type SourceID string

type SourceType string

const (
	SourceWire       SourceType = "wire"
	SourceBroadcast  SourceType = "broadcast"
	SourceNewspaper  SourceType = "newspaper"
	SourcePortal     SourceType = "portal"
	SourceGovernment SourceType = "government"
	SourceNGO        SourceType = "ngo"
)

type Source struct {
	ID       SourceID
	Name     string
	Country  CountryCode
	Region   string
	Language LanguageCode
	Domain   string
	Type     SourceType

	RSSURL     string
	SitemapURL string

	CrawlPolicy string
	Reliability string

	Active       bool
	UpdateFreq   time.Duration
	RegisteredAt time.Time
	UpdatedAt    time.Time
}

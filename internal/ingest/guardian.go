package ingest

import "wend.press/internal/domain"

// NewGuardian returns the Guardian World news feed adapter.
func NewGuardian(cfg CollectorConfig) *RSSAdapter {
	return NewRSSAdapter(domain.Source{
		ID:       "guardian",
		Name:     "The Guardian",
		Domain:   "theguardian.com",
		Country:  "GB",
		Region:   "Europe",
		Language: "en",
		Type:     domain.SourceNewspaper,
		RSSURL:   "https://www.theguardian.com/world/rss",
		Active:   true,
	}, cfg)
}

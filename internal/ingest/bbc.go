package ingest

import "wend.press/internal/domain"

func NewBBC(cfg CollectorConfig) *RSSAdapter {
	return NewRSSAdapter(domain.Source{
		ID:       "bbc-news",
		Name:     "BBC News",
		Domain:   "bbc.co.uk",
		Country:  "GB",
		Region:   "Europe",
		Language: "en",
		Type:     domain.SourceBroadcast,
		RSSURL:   "https://feeds.bbci.co.uk/news/world/rss.xml",
		Active:   true,
	}, cfg)
}

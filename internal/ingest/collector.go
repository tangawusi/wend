// Package ingest owns the crawling layer: Colly configuration, source
// adapters, and the runner that turns discovered URLs into articles.
package ingest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gocolly/colly/v2"
)

const ParserVersion = "ingest/v0.1.0"

type CollectorConfig struct {
	UserAgent    string
	Parallelism  int
	Delay        time.Duration
	Timeout      time.Duration
	MaxBodyBytes int
	IgnoreRobots bool
}

func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		UserAgent:    "wend.press/0.1 (+https://wend.press/bot)",
		Parallelism:  2,
		Delay:        500 * time.Millisecond,
		Timeout:      20 * time.Second,
		MaxBodyBytes: 8 << 20, // 8 MiB
		IgnoreRobots: false,
	}
}

// NewCollector returns a synchronous Colly collector.
//
// colly v2 exposes robots handling and timeouts as struct fields or
// setter methods depending on the option; we use whichever the field
// is for the ones that are fields. Retries are not configured here:
// the runner treats a failed URL as a per-item failure and moves on,
// which is the correct granularity for a news crawl.
func NewCollector(cfg CollectorConfig) (*colly.Collector, error) {
	c := colly.NewCollector(
		colly.UserAgent(cfg.UserAgent),
		colly.MaxBodySize(cfg.MaxBodyBytes),
	)

	c.IgnoreRobotsTxt = cfg.IgnoreRobots

	c.SetRequestTimeout(cfg.Timeout)
	c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	})

	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Parallelism,
		Delay:       cfg.Delay,
		RandomDelay: cfg.Delay / 2,
	}); err != nil {
		return nil, err
	}
	return c, nil
}

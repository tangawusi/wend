package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type SourceInput struct {
	Name       string
	Domain     string
	Country    string
	Region     string
	Language   string
	Type       string
	RSSURL     string
	SitemapURL string
}

func (s *Store) UpsertSource(ctx context.Context, in SourceInput) (uuid.UUID, error) {
	const q = `
		INSERT INTO sources
			(name, domain, country, region, language, type, rss_url, sitemap_url, active)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), NULLIF($8,''), true)
		ON CONFLICT (domain) DO UPDATE SET
			name        = EXCLUDED.name,
			country     = EXCLUDED.country,
			region      = EXCLUDED.region,
			language    = EXCLUDED.language,
			type        = EXCLUDED.type,
			rss_url     = COALESCE(EXCLUDED.rss_url,     sources.rss_url),
			sitemap_url = COALESCE(EXCLUDED.sitemap_url, sources.sitemap_url),
			updated_at  = now()
		RETURNING id`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, q,
		in.Name, in.Domain, in.Country, in.Region,
		in.Language, in.Type, in.RSSURL, in.SitemapURL,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert source: %w", err)
	}
	return id, nil
}

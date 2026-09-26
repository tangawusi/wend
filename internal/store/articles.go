package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ArticleInput struct {
	SourceID      uuid.UUID
	URL           string
	CanonicalURL  string
	Title         string
	Body          string
	Author        string
	Section       string
	PublishedAt   *time.Time
	Language      string
	Country       string
	ContentHash   string
	ParserVersion string
	ModelVersion  string
}

// SaveArticle inserts a new article or returns the existing id when
// (source_id, url) already exists. Updates are handled by Phase 4's
// version-tracking logic; for now an unchanged URL is a no-op.
func (s *Store) SaveArticle(ctx context.Context, in ArticleInput) (uuid.UUID, error) {
	const q = `
		INSERT INTO articles
			(source_id, url, canonical_url, title, body, author, section,
			 published_at, language, publication_country, content_hash,
			 parser_version, model_version)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''),
		        $8, $9, $10, $11, $12, NULLIF($13,''))
		ON CONFLICT (source_id, url) DO UPDATE
			SET updated_at = articles.updated_at
		RETURNING id`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, q,
		in.SourceID, in.URL, in.CanonicalURL, in.Title, in.Body,
		in.Author, in.Section, in.PublishedAt,
		in.Language, in.Country, in.ContentHash,
		in.ParserVersion, in.ModelVersion,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("save article: %w", err)
	}
	return id, nil
}

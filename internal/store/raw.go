package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type RawArticleInput struct {
	SourceID      uuid.UUID
	URL           string
	HTTPStatus    int
	ContentType   string
	Body          []byte
	ContentHash   string
	ParserVersion string
}

func (s *Store) SaveRawArticle(ctx context.Context, in RawArticleInput) (uuid.UUID, error) {
	const q = `
		INSERT INTO raw_articles
			(source_id, url, http_status, content_type, body, content_hash, parser_version)
		VALUES ($1, $2, $3, NULLIF($4,''), $5, $6, $7)
		ON CONFLICT (url, content_hash) DO NOTHING
		RETURNING id`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, q,
		in.SourceID, in.URL, in.HTTPStatus, in.ContentType,
		in.Body, in.ContentHash, in.ParserVersion,
	).Scan(&id)

	if err == nil {
		return id, nil
	}
	if isNoRows(err) {
		const sel = `SELECT id FROM raw_articles WHERE url = $1 AND content_hash = $2`
		if err := s.pool.QueryRow(ctx, sel, in.URL, in.ContentHash).Scan(&id); err != nil {
			return uuid.Nil, fmt.Errorf("select existing raw article: %w", err)
		}
		return id, nil
	}
	return uuid.Nil, fmt.Errorf("save raw article: %w", err)
}

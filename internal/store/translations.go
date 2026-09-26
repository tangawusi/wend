package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TranslationInput struct {
	ArticleID    uuid.UUID
	ToLanguage   string
	FromLanguage string
	Body         string
	Model        string
	ModelVersion string
	SourceHash   string
	OutputHash   string
}

func (s *Store) SaveTranslation(ctx context.Context, in TranslationInput) error {
	const q = `
		INSERT INTO article_translations
			(article_id, to_language, from_language, body,
			 model, model_version, source_hash, output_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (article_id, to_language) DO UPDATE SET
			from_language = EXCLUDED.from_language,
			body          = EXCLUDED.body,
			model         = EXCLUDED.model,
			model_version = EXCLUDED.model_version,
			source_hash   = EXCLUDED.source_hash,
			output_hash   = EXCLUDED.output_hash,
			translated_at = now()`
	if _, err := s.pool.Exec(ctx, q,
		in.ArticleID, in.ToLanguage, in.FromLanguage, in.Body,
		in.Model, in.ModelVersion, in.SourceHash, in.OutputHash,
	); err != nil {
		return fmt.Errorf("save translation: %w", err)
	}
	return nil
}

func (s *Store) GetTranslation(ctx context.Context, articleID uuid.UUID, toLanguage string) (string, error) {
	const q = `SELECT body FROM article_translations WHERE article_id = $1 AND to_language = $2`
	var body string
	err := s.pool.QueryRow(ctx, q, articleID, toLanguage).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get translation: %w", err)
	}
	return body, nil
}

package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type EntityInput struct {
	Kind    string
	Name    string
	Country string
}

type ArticleEntityInput struct {
	ArticleID uuid.UUID
	EntityID  uuid.UUID
	Mentions  int
	Salience  float32
}

func (s *Store) UpsertEntity(ctx context.Context, in EntityInput) (uuid.UUID, error) {
	const q = `
		INSERT INTO entities (kind, canonical, country)
		VALUES ($1, $2, NULLIF($3,''))
		ON CONFLICT (kind, canonical) DO UPDATE
			SET country = COALESCE(EXCLUDED.country, entities.country)
		RETURNING id`
	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, q, in.Kind, in.Name, in.Country).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert entity: %w", err)
	}
	return id, nil
}

func (s *Store) LinkArticleEntity(ctx context.Context, in ArticleEntityInput) error {
	const q = `
		INSERT INTO article_entities (article_id, entity_id, mentions, salience)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (article_id, entity_id) DO UPDATE
			SET mentions = EXCLUDED.mentions,
			    salience = EXCLUDED.salience`
	if _, err := s.pool.Exec(ctx, q, in.ArticleID, in.EntityID, in.Mentions, in.Salience); err != nil {
		return fmt.Errorf("link article entity: %w", err)
	}
	return nil
}

// ResetEntities deletes all entity rows and their edges. Used when the
// extraction pipeline changes enough that prior rows are no longer
// trustworthy. Articles and events are untouched; only entity data is
// cleared, and the next crawl repopulates it.
func (s *Store) ResetEntities(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, q := range []string{
		`DELETE FROM event_entities`,
		`DELETE FROM article_entities`,
		`DELETE FROM entities`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("reset entities: %w", err)
		}
	}
	return tx.Commit(ctx)
}

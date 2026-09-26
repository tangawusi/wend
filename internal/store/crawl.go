package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) StartCrawlRun(ctx context.Context, sourceID uuid.UUID, mode string) (uuid.UUID, error) {
	const q = `
		INSERT INTO crawl_runs (source_id, mode, status)
		VALUES ($1, $2, 'running')
		RETURNING id`

	var id uuid.UUID
	if err := s.pool.QueryRow(ctx, q, sourceID, mode).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("start crawl run: %w", err)
	}
	return id, nil
}

func (s *Store) FinishCrawlRun(
	ctx context.Context,
	runID uuid.UUID,
	status string,
	discovered, created, failed int,
	errMsg string,
) error {
	const q = `
		UPDATE crawl_runs
		   SET finished_at      = now(),
		       status           = $2,
		       urls_discovered  = $3,
		       urls_fetched     = $3,
		       articles_created = $4,
		       error            = NULLIF($5,'')
		 WHERE id = $1`

	_, err := s.pool.Exec(ctx, q, runID, status, discovered, created, errMsg)
	return err
}

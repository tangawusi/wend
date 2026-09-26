package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// NearDuplicate describes a candidate match returned by the SimHash
// scan. It is not proof of duplication; the caller decides what to do
// with it.
type NearDuplicate struct {
	ArticleID  uuid.UUID
	Title      string
	URL        string
	SourceName string
	Distance   int
}

// SetSimHash stores an article's fingerprint. Separate from SaveArticle
// because it is idempotent and safe to call on any existing row, which
// makes backfilling existing articles a simple loop.
func (s *Store) SetSimHash(ctx context.Context, articleID uuid.UUID, bits string) error {
	const q = `UPDATE articles SET simhash = $2::bit(64) WHERE id = $1`
	if _, err := s.pool.Exec(ctx, q, articleID, bits); err != nil {
		return fmt.Errorf("set simhash: %w", err)
	}
	return nil
}

// FindNearDuplicates returns articles whose SimHash is within
// maxDistance of the given fingerprint, restricted to the last
// withinDays days and excluding the article itself.
//
// The query cannot use an index on simhash: popcount is not indexable.
// It is bounded by the time window instead. At v1 scale that is fine;
// the production path is banded LSH (split the 64 bits into 4×16-bit
// bands, index each, union candidates, verify) when the window stops
// being small enough.
func (s *Store) FindNearDuplicates(
	ctx context.Context,
	bits string,
	maxDistance int,
	withinDays int,
	excludeArticleID uuid.UUID,
) ([]NearDuplicate, error) {
	const q = `
		SELECT a.id, a.title, a.url, s.name,
		       bit_count(a.simhash # $1::bit(64))::int AS distance
		  FROM articles a
		  JOIN sources  s ON s.id = a.source_id
		 WHERE a.simhash IS NOT NULL
		   AND a.id <> $2
		   AND a.discovered_at > now() - make_interval(days => $3)
		   AND bit_count(a.simhash # $1::bit(64)) <= $4
		 ORDER BY distance ASC, a.discovered_at DESC
		 LIMIT 50`

	rows, err := s.pool.Query(ctx, q, bits, excludeArticleID, withinDays, maxDistance)
	if err != nil {
		return nil, fmt.Errorf("find near duplicates: %w", err)
	}
	defer rows.Close()

	var out []NearDuplicate
	for rows.Next() {
		var n NearDuplicate
		if err := rows.Scan(&n.ArticleID, &n.Title, &n.URL, &n.SourceName, &n.Distance); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

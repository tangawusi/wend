package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ArticleForClustering is the projection the clusterer consumes. It is
// deliberately not the full Article: clustering needs title, body,
// country, time, and entities, and nothing else.
type ArticleForClustering struct {
	ID         uuid.UUID
	Title      string
	Body       string
	Language   string
	Country    string
	ObservedAt time.Time
	EntityIDs  []uuid.UUID
}

// CandidateEvent is an event that could absorb a new article. EntityIDs
// is a set rather than a slice because every use is membership testing.
type CandidateEvent struct {
	ID            uuid.UUID
	Title         string
	Body          string
	Country       string
	EntityIDs     map[uuid.UUID]struct{}
	LastObserved  time.Time
	ArticleCount  int
}

func (s *Store) FindUnclusteredArticles(ctx context.Context, limit int) ([]ArticleForClustering, error) {
	const q = `
		SELECT id, title, body, language, publication_country,
		       COALESCE(published_at, discovered_at) AS observed_at
		  FROM articles
		 WHERE clustered_at IS NULL
		 ORDER BY COALESCE(published_at, discovered_at) ASC
		 LIMIT $1`

	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("find unclustered: %w", err)
	}
	defer rows.Close()

	var out []ArticleForClustering
	for rows.Next() {
		var a ArticleForClustering
		if err := rows.Scan(&a.ID, &a.Title, &a.Body, &a.Language, &a.Country, &a.ObservedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LoadArticleEntities returns entity ids per article for the given
// article ids. One query regardless of count.
func (s *Store) LoadArticleEntities(ctx context.Context, articleIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	if len(articleIDs) == 0 {
		return map[uuid.UUID][]uuid.UUID{}, nil
	}
	const q = `
		SELECT article_id, entity_id
		  FROM article_entities
		 WHERE article_id = ANY($1)`

	rows, err := s.pool.Query(ctx, q, articleIDs)
	if err != nil {
		return nil, fmt.Errorf("load article entities: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]uuid.UUID, len(articleIDs))
	for rows.Next() {
		var aid, eid uuid.UUID
		if err := rows.Scan(&aid, &eid); err != nil {
			return nil, err
		}
		out[aid] = append(out[aid], eid)
	}
	return out, rows.Err()
}

// LoadCandidateEvents returns events whose last observation is within
// the window, along with their aggregated entity sets and a
// representative body (the earliest article's).
func (s *Store) LoadCandidateEvents(ctx context.Context, since time.Time) ([]CandidateEvent, error) {
	const q = `
		SELECT e.id,
		       e.title,
		       COALESCE(rep.body, '') AS body,
		       COALESCE(rep.country, '') AS country,
		       e.last_observed_time,
		       e.article_count,
		       COALESCE(
		           array_agg(DISTINCT ee.entity_id)
		               FILTER (WHERE ee.entity_id IS NOT NULL),
		           ARRAY[]::uuid[]
		       ) AS entity_ids
		  FROM events e
		  LEFT JOIN event_entities ee ON ee.event_id = e.id
		  LEFT JOIN LATERAL (
		      SELECT a.body, a.publication_country AS country
		        FROM event_articles ea
		        JOIN articles a ON a.id = ea.article_id
		       WHERE ea.event_id = e.id
		       ORDER BY COALESCE(a.published_at, a.discovered_at) ASC
		       LIMIT 1
		  ) rep ON true
		 WHERE e.last_observed_time >= $1
		 GROUP BY e.id, e.title, rep.body, rep.country,
		          e.last_observed_time, e.article_count`

	rows, err := s.pool.Query(ctx, q, since)
	if err != nil {
		return nil, fmt.Errorf("load candidate events: %w", err)
	}
	defer rows.Close()

	var out []CandidateEvent
	for rows.Next() {
		var ev CandidateEvent
		var ids []uuid.UUID
		if err := rows.Scan(
			&ev.ID, &ev.Title, &ev.Body, &ev.Country,
			&ev.LastObserved, &ev.ArticleCount, &ids,
		); err != nil {
			return nil, err
		}
		ev.EntityIDs = make(map[uuid.UUID]struct{}, len(ids))
		for _, id := range ids {
			ev.EntityIDs[id] = struct{}{}
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// CreateEventFromArticle inserts a new event, links the article to it,
// and marks the article clustered — all in one transaction, so a crash
// cannot leave an orphan event or an eventless clustered article.
func (s *Store) CreateEventFromArticle(ctx context.Context, a ArticleForClustering, modelVersion string) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	const evQ = `
		INSERT INTO events
			(title, summary, state, start_time, last_observed_time,
			 article_count, source_count, language_count,
			 cluster_confidence, model_version)
		VALUES ($1, $2, 'emerging', $3, $3, 1, 1, 1, 1.0, $4)
		RETURNING id`

	summary := truncateRunes(a.Body, 500)

	var eventID uuid.UUID
	if err := tx.QueryRow(ctx, evQ, a.Title, summary, a.ObservedAt, modelVersion).Scan(&eventID); err != nil {
		return uuid.Nil, fmt.Errorf("insert event: %w", err)
	}

	const eaQ = `
		INSERT INTO event_articles (event_id, article_id, similarity)
		VALUES ($1, $2, 1.0)
		ON CONFLICT (event_id, article_id) DO NOTHING`
	if _, err := tx.Exec(ctx, eaQ, eventID, a.ID); err != nil {
		return uuid.Nil, fmt.Errorf("link article: %w", err)
	}

	const markQ = `UPDATE articles SET clustered_at = now() WHERE id = $1`
	if _, err := tx.Exec(ctx, markQ, a.ID); err != nil {
		return uuid.Nil, fmt.Errorf("mark clustered: %w", err)
	}

	const mergeQ = `
		INSERT INTO event_entities (event_id, entity_id, role)
		SELECT $1, ae.entity_id, 'mentioned'
		  FROM article_entities ae
		 WHERE ae.article_id = $2
		ON CONFLICT (event_id, entity_id) DO NOTHING`
	if _, err := tx.Exec(ctx, mergeQ, eventID, a.ID); err != nil {
		return uuid.Nil, fmt.Errorf("merge entities: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return eventID, nil
}

// AssignArticleToEvent links an article to an existing event and marks
// it clustered. Idempotent on (event_id, article_id).
func (s *Store) AssignArticleToEvent(ctx context.Context, eventID, articleID uuid.UUID, similarity float32) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const eaQ = `
		INSERT INTO event_articles (event_id, article_id, similarity)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id, article_id) DO UPDATE
			SET similarity = EXCLUDED.similarity`
	if _, err := tx.Exec(ctx, eaQ, eventID, articleID, similarity); err != nil {
		return fmt.Errorf("link article: %w", err)
	}

	const markQ = `UPDATE articles SET clustered_at = now() WHERE id = $1`
	if _, err := tx.Exec(ctx, markQ, articleID); err != nil {
		return fmt.Errorf("mark clustered: %w", err)
	}

	return tx.Commit(ctx)
}

// MergeEventEntities copies an article's entity edges onto the event.
func (s *Store) MergeEventEntities(ctx context.Context, eventID, articleID uuid.UUID) error {
	const q = `
		INSERT INTO event_entities (event_id, entity_id, role)
		SELECT $1, ae.entity_id, 'mentioned'
		  FROM article_entities ae
		 WHERE ae.article_id = $2
		ON CONFLICT (event_id, entity_id) DO NOTHING`
	if _, err := s.pool.Exec(ctx, q, eventID, articleID); err != nil {
		return fmt.Errorf("merge event entities: %w", err)
	}
	return nil
}

// RefreshEventStats recomputes denormalized counts from event_articles.
func (s *Store) RefreshEventStats(ctx context.Context, eventID uuid.UUID) error {
	const q = `
		UPDATE events e
		   SET article_count      = sub.n,
		       source_count       = sub.s,
		       language_count     = sub.l,
		       last_observed_time = sub.last,
		       updated_at         = now()
		  FROM (
		      SELECT count(*)                          AS n,
		             count(DISTINCT a.source_id)       AS s,
		             count(DISTINCT a.language)        AS l,
		             max(COALESCE(a.published_at, a.discovered_at)) AS last
		        FROM event_articles ea
		        JOIN articles a ON a.id = ea.article_id
		       WHERE ea.event_id = $1
		  ) sub
		 WHERE e.id = $1`
	if _, err := s.pool.Exec(ctx, q, eventID); err != nil {
		return fmt.Errorf("refresh event stats: %w", err)
	}
	return nil
}

// UpdateEventState sets the lifecycle state based on the event's
// current article count and last observation time.
func (s *Store) UpdateEventState(ctx context.Context, eventID uuid.UUID, now time.Time) error {
	const q = `
		UPDATE events
		   SET state = CASE
		           WHEN article_count <= 1 THEN 'emerging'
		           WHEN $2 - last_observed_time < interval '1 hour'  THEN 'active'
		           WHEN $2 - last_observed_time < interval '6 hours' THEN 'developing'
		           WHEN $2 - last_observed_time < interval '24 hours' THEN 'cooling'
		           WHEN $2 - last_observed_time < interval '72 hours' THEN 'resolved'
		           ELSE 'stale'
		       END,
		       updated_at = now()
		 WHERE id = $1`
	if _, err := s.pool.Exec(ctx, q, eventID, now); err != nil {
		return fmt.Errorf("update event state: %w", err)
	}
	return nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

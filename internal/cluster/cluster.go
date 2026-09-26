package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"wend.press/internal/store"
)

// Threshold is the minimum weighted score for an article to be
// assigned to an existing event rather than starting a new one.
// Tuned conservatively: a false merge is worse than a false split,
// because a merge is invisible to the user and a split is correctable.
const Threshold = 0.45

// Window bounds how far back candidate events are considered. A wire
// story republished after this window is a new event, not a duplicate.
const Window = 7 * 24 * time.Hour

// MaxArticlesPerPass bounds one clustering run. The pass is idempotent;
// running it repeatedly drains a backlog.
const MaxArticlesPerPass = 500

type Clusterer struct {
	store    *store.Store
	weights  Weights
	window   time.Duration
	threshold float64
	max      int
}

func NewClusterer(s *store.Store) *Clusterer {
	return &Clusterer{
		store:     s,
		weights:   DefaultWeights(),
		window:    Window,
		threshold: Threshold,
		max:       MaxArticlesPerPass,
	}
}

// Run processes all unclustered articles in chronological order.
// Chronological ordering matters: it makes the pass deterministic and
// means an event's first article is genuinely its earliest, not
// whichever the database happened to return first.
func (c *Clusterer) Run(ctx context.Context) error {
	articles, err := c.store.FindUnclusteredArticles(ctx, c.max)
	if err != nil {
		return fmt.Errorf("find unclustered: %w", err)
	}
	if len(articles) == 0 {
		slog.Info("cluster: no unclustered articles")
		return nil
	}
	slog.Info("cluster: pass starting", "articles", len(articles))

	ids := make([]uuid.UUID, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}
	entitiesByArticle, err := c.store.LoadArticleEntities(ctx, ids)
	if err != nil {
		return fmt.Errorf("load article entities: %w", err)
	}
	for i := range articles {
		articles[i].EntityIDs = entitiesByArticle[articles[i].ID]
	}

	candidates, err := c.store.LoadCandidateEvents(ctx, time.Now().Add(-c.window))
	if err != nil {
		return fmt.Errorf("load candidates: %w", err)
	}
	slog.Info("cluster: candidates loaded", "events", len(candidates))

	var created, assigned, failed int
	touched := map[uuid.UUID]struct{}{}

	for i := range articles {
		if err := ctx.Err(); err != nil {
			return err
		}
		a := &articles[i]

		bestIdx, bestScore := c.bestMatch(*a, candidates)

		if bestIdx >= 0 && bestScore >= c.threshold {
			ev := &candidates[bestIdx]
			if err := c.store.AssignArticleToEvent(ctx, ev.ID, a.ID, float32(bestScore)); err != nil {
				slog.Warn("assign failed", "article", a.ID, "event", ev.ID, "err", err)
				failed++
				continue
			}
			if err := c.store.MergeEventEntities(ctx, ev.ID, a.ID); err != nil {
				slog.Warn("merge entities", "event", ev.ID, "err", err)
			}
			// Update the in-memory candidate so later articles in this
			// pass see the grown entity set.
			for _, eid := range a.EntityIDs {
				ev.EntityIDs[eid] = struct{}{}
			}
			ev.LastObserved = laterTime(ev.LastObserved, a.ObservedAt)
			ev.ArticleCount++
			assigned++
			touched[ev.ID] = struct{}{}
			slog.Debug("assigned",
				"article", a.ID, "event", ev.ID,
				"score", fmt.Sprintf("%.3f", bestScore),
				"signals", fmt.Sprintf("%+v", c.signalsFor(*a, *ev)))
			continue
		}

		eventID, err := c.store.CreateEventFromArticle(ctx, *a, ModelVersion)
		if err != nil {
			slog.Warn("create event failed", "article", a.ID, "err", err)
			failed++
			continue
		}
		entSet := make(map[uuid.UUID]struct{}, len(a.EntityIDs))
		for _, eid := range a.EntityIDs {
			entSet[eid] = struct{}{}
		}
		candidates = append(candidates, store.CandidateEvent{
			ID:            eventID,
			Title:         a.Title,
			Body:          a.Body,
			Country:       a.Country,
			EntityIDs:     entSet,
			LastObserved:  a.ObservedAt,
			ArticleCount:  1,
		})
		created++
		touched[eventID] = struct{}{}
	}

	now := time.Now()
	for id := range touched {
		if err := c.store.RefreshEventStats(ctx, id); err != nil {
			slog.Warn("refresh stats", "event", id, "err", err)
			continue
		}
		if err := c.store.UpdateEventState(ctx, id, now); err != nil {
			slog.Warn("update state", "event", id, "err", err)
		}
	}

	slog.Info("cluster: pass finished",
		"created", created, "assigned", assigned, "failed", failed,
		"events_touched", len(touched))
	return nil
}

// bestMatch returns the index of the candidate with the highest score,
// or -1 when there are no candidates.
func (c *Clusterer) bestMatch(a store.ArticleForClustering, candidates []store.CandidateEvent) (int, float64) {
	bestIdx := -1
	bestScore := 0.0
	for i := range candidates {
		s := c.signalsFor(a, candidates[i])
		score := c.weights.Score(s)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	return bestIdx, bestScore
}

func (c *Clusterer) signalsFor(a store.ArticleForClustering, ev store.CandidateEvent) Signals {
	articleEnts := make(map[string]struct{}, len(a.EntityIDs))
	for _, id := range a.EntityIDs {
		articleEnts[id.String()] = struct{}{}
	}
	eventEnts := make(map[string]struct{}, len(ev.EntityIDs))
	for id := range ev.EntityIDs {
		eventEnts[id.String()] = struct{}{}
	}
	return Compare(
		a.Title, a.Body, a.Country, articleEnts, a.ObservedAt,
		ev.Title, ev.Body, ev.Country, eventEnts, ev.LastObserved,
	)
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

package cluster

import (
	"math"
	"time"
)

// Signals is the breakdown of one article-to-event comparison. It is
// returned rather than collapsed to a scalar so the scoring is
// inspectable: a low-confidence assignment can be explained.
type Signals struct {
	EntityJaccard float64
	TitleJaccard  float64
	BodyJaccard   float64
	TemporalDecay float64
	GeoMatch      float64
}

// Weights must sum to 1.0. Entity overlap is the strongest single
// signal in news clustering: two articles about the same event
// overwhelmingly name the same people, organizations, and places.
type Weights struct {
	Entity   float64
	Title    float64
	Body     float64
	Temporal float64
	Geo      float64
}

func DefaultWeights() Weights {
	return Weights{
		Entity:   0.35,
		Title:    0.20,
		Body:     0.25,
		Temporal: 0.15,
		Geo:      0.05,
	}
}

// Score collapses Signals into a single value in [0,1].
func (w Weights) Score(s Signals) float64 {
	return s.EntityJaccard*w.Entity +
		s.TitleJaccard*w.Title +
		s.BodyJaccard*w.Body +
		s.TemporalDecay*w.Temporal +
		s.GeoMatch*w.Geo
}

// temporalHalfLife is the interval over which a time difference halves
// the temporal signal. 48 hours captures the window in which news
// organizations report on the same occurrence; beyond a week the
// signal is negligible.
const temporalHalfLife = 48 * time.Hour

// TemporalDecay returns a value in (0,1] falling off exponentially
// with the gap between two timestamps. Order-independent.
func TemporalDecay(a, b time.Time) float64 {
	gap := a.Sub(b)
	if gap < 0 {
		gap = -gap
	}
	return math.Exp(-float64(gap) / float64(temporalHalfLife))
}

// GeoMatch scores two country codes. Same country is a weak positive;
// a mismatch is not negative, because an event in one country is
// routinely reported by outlets in another.
func GeoMatch(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	return 0
}

// Compare computes the full signal breakdown for one article against
// one event's representative text and entity set.
func Compare(
	articleTitle, articleBody, articleCountry string,
	articleEntities map[string]struct{},
	articleTime time.Time,
	eventTitle, eventBody, eventCountry string,
	eventEntities map[string]struct{},
	eventTime time.Time,
) Signals {
	return Signals{
		EntityJaccard: Jaccard(articleEntities, eventEntities),
		TitleJaccard:  Jaccard(TokenSet(articleTitle, 0), TokenSet(eventTitle, 0)),
		BodyJaccard:   Jaccard(TokenSet(articleBody, maxBodyTokens), TokenSet(eventBody, maxBodyTokens)),
		TemporalDecay: TemporalDecay(articleTime, eventTime),
		GeoMatch:      GeoMatch(articleCountry, eventCountry),
	}
}

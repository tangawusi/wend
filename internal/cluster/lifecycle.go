package cluster

import "time"

// Lifecycle thresholds. An event's state is derived from how recently
// it was observed and how many independent articles corroborate it.
// These are product decisions, not constants of the universe; the
// values below are a starting point to be tuned against real data.
const (
	activeWindow     = 1 * time.Hour
	developingWindow = 6 * time.Hour
	coolingWindow    = 24 * time.Hour
	resolvedWindow   = 72 * time.Hour
)

// StateFor derives an event's lifecycle state.
//
// A single-article event is "emerging" regardless of age: one report
// is a claim, not an event. Corroboration is what advances an event
// past emerging.
func StateFor(articleCount int, lastObserved time.Time, now time.Time) string {
	if articleCount <= 1 {
		return "emerging"
	}
	age := now.Sub(lastObserved)
	switch {
	case age < activeWindow:
		return "active"
	case age < developingWindow:
		return "developing"
	case age < coolingWindow:
		return "cooling"
	case age < resolvedWindow:
		return "resolved"
	default:
		return "stale"
	}
}

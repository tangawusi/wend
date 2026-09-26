package domain

import "time"

type EventID string

type EventState string

const (
	EventEmerging   EventState = "emerging"
	EventActive     EventState = "active"
	EventDeveloping EventState = "developing"
	EventCooling    EventState = "cooling"
	EventResolved   EventState = "resolved"
	EventStale      EventState = "stale"
)

type Event struct {
	ID      EventID
	Title   string
	Summary string

	Location *GeoPoint

	StartTime        time.Time
	LastObservedTime time.Time

	State EventState

	ArticleCount  int
	SourceCount   int
	LanguageCount int

	Provenance Provenance
}

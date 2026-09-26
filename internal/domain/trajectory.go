package domain

import "time"

type TrajectoryID string

type Direction string

const (
	DirectionEscalating   Direction = "escalating"
	DirectionStable       Direction = "stable"
	DirectionDeescalating Direction = "deescalating"
	DirectionUnclear      Direction = "unclear"
)

type CoverageTier string

const (
	CoverageThin     CoverageTier = "thin"
	CoverageModerate CoverageTier = "moderate"
	CoverageHigh     CoverageTier = "high"
)

type Coverage struct {
	Tier                CoverageTier
	SourceCount         int
	SourceDiversity     float64
	CountryDiversity    int
	LanguageDiversity   int
	TemporalPersistence time.Duration
}

type Confidence struct {
	Score       float64
	Explanation []ConfidenceFactor
}

type ConfidenceFactor struct {
	Name   string
	Weight float64
	Value  float64
	Note   string
}

type Trajectory struct {
	ID      TrajectoryID
	EventID EventID

	ObservedAt time.Time

	Direction Direction
	Momentum  float64

	Confidence Confidence
	Coverage   Coverage

	ModelVersion string

	Provenance Provenance
}

package domain

import "time"

type Provenance struct {
	CreatedAt time.Time
	UpdatedAt time.Time

	ContentHash string

	DerivedFrom []string

	ParserVersion string
	ModelVersion  string

	AnchorRef string
}

package domain

import "time"

type ArticleID string

type Article struct {
	ID       ArticleID
	SourceID SourceID

	URL          string
	CanonicalURL string

	Title string
	Body  string

	TranslatedBody string

	Author  string
	Section string

	PublishedAt  *time.Time
	DiscoveredAt time.Time

	Language           LanguageCode
	PublicationCountry CountryCode

	ContentHash string

	Provenance Provenance
}

package domain

import "time"

type LanguageCode string
type CountryCode string

type Translation struct {
	ArticleID    ArticleID
	FromLanguage LanguageCode
	ToLanguage   LanguageCode
	Model        string
	ModelVersion string
	TranslatedAt time.Time
	SourceHash   string
	OutputHash   string
}

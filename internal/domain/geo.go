package domain

type GeoPoint struct {
	Country    CountryCode
	Region     string
	City       string
	Confidence float64
}

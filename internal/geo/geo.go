package geo

import (
	"strings"
	"sync"
)

type Country struct {
	Alpha2 string
	Alpha3 string
	Name   string
	Region string
}

type Registry struct {
	once   sync.Once
	byName map[string]*Country
	byA2   map[string]*Country
	byA3   map[string]*Country
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) build() {
	r.once.Do(func() {
		n := len(countryData)
		r.byName = make(map[string]*Country, n*2)
		r.byA2 = make(map[string]*Country, n)
		r.byA3 = make(map[string]*Country, n)

		for i := range countryData {
			c := &countryData[i]
			rec := &Country{
				Alpha2: c.Alpha2,
				Alpha3: c.Alpha3,
				Name:   c.Name,
				Region: c.Region,
			}
			r.byA2[rec.Alpha2] = rec
			r.byA3[rec.Alpha3] = rec
			r.byName[strings.ToLower(rec.Name)] = rec
		}

		aliases := map[string]string{
			"uk": "GB", "britain": "GB", "great britain": "GB",
			"england": "GB", "scotland": "GB", "wales": "GB",
			"northern ireland": "GB",
			"usa": "US", "united states of america": "US", "america": "US",
			"u.s.": "US", "u.s.a.": "US", "the united states": "US",
			"russia": "RU", "russian federation": "RU",
			"iran": "IR", "islamic republic of iran": "IR",
			"vietnam": "VN", "viet nam": "VN",
			"syria": "SY", "syrian arab republic": "SY",
			"bolivia": "BO", "plurinational state of bolivia": "BO",
			"venezuela": "VE", "bolivarian republic of venezuela": "VE",
			"taiwan": "TW", "republic of china": "TW",
			"czech republic": "CZ", "czechia": "CZ",
			"ivory coast": "CI", "côte d'ivoire": "CI", "cote d'ivoire": "CI",
			"drc": "CD", "democratic republic of the congo": "CD",
			"republic of the congo": "CG", "congo-brazzaville": "CG",
			"congo-kinshasa": "CD",
			"uae": "AE", "united arab emirates": "AE",
			"burma": "MM", "myanmar": "MM",
			"south korea": "KR", "republic of korea": "KR",
			"north korea": "KP", "dprk": "KP",
			"holland": "NL", "the netherlands": "NL",
			"macedonia": "MK",
			"swaziland": "SZ", "eswatini": "SZ",
			"east timor": "TL", "timor-leste": "TL",
			"cape verde": "CV", "cabo verde": "CV",
		}
		for alias, a2 := range aliases {
			if rec, ok := r.byA2[a2]; ok {
				r.byName[alias] = rec
			}
		}
	})
}

func (r *Registry) Resolve(s string) (*Country, bool) {
	r.build()
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	if c, ok := r.byA2[strings.ToUpper(s)]; ok {
		return c, true
	}
	if c, ok := r.byA3[strings.ToUpper(s)]; ok {
		return c, true
	}
	if c, ok := r.byName[strings.ToLower(s)]; ok {
		return c, true
	}
	return nil, false
}

func (r *Registry) ISO2OrEmpty(s string) string {
	if c, ok := r.Resolve(s); ok {
		return c.Alpha2
	}
	return ""
}

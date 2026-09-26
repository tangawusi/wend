// Package entity extracts named entities from article text.
//
// The extractor is prose/v3. Its quality on news text is limited —
// photo credits, agency acronyms, and boilerplate routinely appear as
// entities — so this package does what it can to reject junk before
// anything reaches the schema. The model is the ceiling; the filters
// here keep the floor from being worse than the model.
package entity

import (
	"strings"
	"unicode"

	"github.com/tsawler/prose/v3"
)

// Kind matches the entities.kind CHECK constraint, with one addition:
// KindGeo is an internal marker for a GPE mention the caller must
// resolve into country, city, or location. It is never stored.
type Kind string

const (
	KindPerson       Kind = "person"
	KindOrganization Kind = "organization"
	KindCountry      Kind = "country"    // resolved by caller
	KindCity         Kind = "city"       // resolved by caller
	KindGeo          Kind = "geo"        // unresolved GPE; caller must resolve
	KindLocation     Kind = "location"
	KindProduct      Kind = "product"
	KindEvent        Kind = "event"
)

type Extracted struct {
	Text     string
	Kind     Kind
	Mentions int
	Salience float32
}

type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

func (e *Extractor) Extract(title, body string) []Extracted {
	text := title + "\n\n" + body
	if strings.TrimSpace(text) == "" {
		return nil
	}

	doc, err := prose.NewDocument(text)
	if err != nil {
		return nil
	}

	counts := map[string]*Extracted{}
	for _, ent := range doc.Entities() {
		raw := strings.TrimSpace(ent.Text)
		if isNoise(raw) {
			continue
		}
		kind, ok := labelToKind(ent.Label)
		if !ok {
			continue
		}
		key := strings.ToLower(raw) + "\x1f" + string(kind)
		if existing, ok := counts[key]; ok {
			existing.Mentions++
			continue
		}
		counts[key] = &Extracted{
			Text:     raw,
			Kind:     kind,
			Mentions: 1,
		}
	}

	if len(counts) == 0 {
		return nil
	}

	out := make([]Extracted, 0, len(counts))
	for _, ex := range counts {
		out = append(out, *ex)
	}

	max := 1
	for _, ex := range out {
		if ex.Mentions > max {
			max = ex.Mentions
		}
	}
	for i := range out {
		out[i].Salience = float32(out[i].Mentions) / float32(max)
	}
	return out
}

// isNoise rejects entity spans that prose produced but that cannot be
// entities in a news product: photo credits, agency boilerplate,
// paragraph fragments, and punctuation-edged tokens.
func isNoise(s string) bool {
	if s == "" {
		return true
	}

	// Photo credits contain a slash. So do some legitimate compound
	// names, but in news prose the false-positive rate of rejecting
	// them is far lower than the false-negative rate of accepting
	// "AFP/Getty Images Members" as a person.
	if strings.ContainsRune(s, '/') {
		return true
	}

	lower := strings.ToLower(s)

	// Boilerplate tokens. These words do not appear in the canonical
	// form of any entity worth storing.
	for _, tok := range []string{
		"getty", "shutterstock", "anadolu", "associated press",
		" photo ", " image ", " images", " credit ",
	} {
		if strings.Contains(lower, tok) || strings.HasPrefix(lower, strings.TrimSpace(tok)+" ") {
			return true
		}
	}
	// Handle prefix/suffix at string boundaries separately.
	for _, tok := range []string{"getty", "shutterstock", "images", "image", "photo"} {
		if strings.HasPrefix(lower, tok) || strings.HasSuffix(lower, tok) {
			return true
		}
	}

	// Very short single tokens are usually fragments.
	if len([]rune(s)) < 2 {
		return true
	}

	// More than 8 words is a sentence fragment, not a name.
	if len(strings.Fields(s)) > 8 {
		return true
	}

	// Over 80 runes is a paragraph fragment.
	if len([]rune(s)) > 80 {
		return true
	}

	// Starting or ending with punctuation indicates a truncated span
	// or a possessive that prose mis-segmented.
	runes := []rune(s)
	first, last := runes[0], runes[len(runes)-1]
	if !unicode.IsLetter(first) && !unicode.IsDigit(first) {
		return true
	}
	if !unicode.IsLetter(last) && !unicode.IsDigit(last) {
		return true
	}

	return false
}

// labelToKind maps prose's entity labels to schema kinds. GPE maps to
// KindGeo — not KindCountry — because GPE means "geopolitical entity"
// and includes cities, regions, continents, and national bodies. The
// caller resolves KindGeo against the country registry and gazetteer.
func labelToKind(label string) (Kind, bool) {
	switch strings.ToUpper(label) {
	case "PERSON":
		return KindPerson, true
	case "ORG":
		return KindOrganization, true
	case "GPE":
		return KindGeo, true
	case "FAC":
		return KindLocation, true
	case "PRODUCT":
		return KindProduct, true
	case "EVENT":
		return KindEvent, true
	case "NORP":
		return KindOrganization, true
	}
	return "", false
}

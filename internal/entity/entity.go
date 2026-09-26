package entity

import (
	"strings"

	"github.com/tsawler/prose/v3"
)

type Kind string

const (
	KindPerson       Kind = "person"
	KindOrganization Kind = "organization"
	KindCountry      Kind = "country"
	KindCity         Kind = "city"
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
		kind, ok := labelToKind(ent.Label)
		if !ok {
			continue
		}
		key := strings.ToLower(ent.Text) + "\x1f" + string(kind)
		if existing, ok := counts[key]; ok {
			existing.Mentions++
			continue
		}
		counts[key] = &Extracted{
			Text:     strings.TrimSpace(ent.Text),
			Kind:     kind,
			Mentions: 1,
		}
	}

	if len(counts) == 0 {
		return nil
	}

	out := make([]Extracted, 0, len(counts))
	for _, ex := range counts {
		if len([]rune(ex.Text)) < 2 {
			continue
		}
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

func labelToKind(label string) (Kind, bool) {
	switch strings.ToUpper(label) {
	case "PERSON":
		return KindPerson, true
	case "ORG":
		return KindOrganization, true
	case "GPE":
		return KindCountry, true
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

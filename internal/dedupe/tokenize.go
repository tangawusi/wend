// Package dedupe implements exact and near-duplicate detection over
// articles. It is the layer that recognizes when two publications are
// reporting the same wire story rather than independent events.
//
// The fingerprint is SimHash64 over content tokens. Near-duplicates
// are those within a small Hamming distance; the exact threshold is
// a tuning surface, not a constant of the universe.
package dedupe

import (
	"strings"
	"unicode"
)

// stopwords is a small multilingual list. It is not exhaustive: SimHash
// is robust to a few surviving function words, and a longer list would
// introduce language-maintenance burden without changing outcomes.
// The real per-language tokenizer belongs to the language layer (§8).
var stopwords = map[string]struct{}{
	// English
	"the": {}, "and": {}, "of": {}, "to": {}, "in": {}, "is": {}, "for": {},
	"on": {}, "that": {}, "by": {}, "this": {}, "with": {}, "it": {}, "as": {},
	"are": {}, "be": {}, "was": {}, "from": {}, "at": {}, "or": {}, "an": {},
	"have": {}, "has": {}, "had": {}, "not": {}, "but": {}, "his": {}, "her": {},
	// French
	"le": {}, "la": {}, "les": {}, "de": {}, "des": {}, "du": {}, "un": {}, "une": {},
	"et": {}, "est": {}, "en": {}, "que": {}, "qui": {}, "dans": {}, "pour": {}, "par": {},
	// Spanish
	"el": {}, "los": {}, "las": {}, "del": {}, "una": {}, "y": {}, "por": {},
	"con": {}, "para": {}, "sus": {}, "como": {},
	// German
	"der": {}, "die": {}, "das": {}, "und": {}, "von": {}, "zu": {}, "den": {},
	"dem": {}, "ein": {}, "eine": {}, "ist": {}, "auf": {}, "mit": {}, "sich": {},
	// Portuguese
	"os": {}, "do": {}, "da": {}, "dos": {}, "um": {}, "uma": {}, "e": {},
	"em": {}, "seu": {}, "sua": {},
	// Italian
	"il": {}, "lo": {}, "gli": {}, "di": {}, "che": {}, "per": {},
}

// Tokenize splits text into content tokens: lowercased, filtered of
// stopwords and short words, with CJK split into individual runes.
//
// CJK has no whitespace word boundaries; a per-character split is the
// simplest scheme that yields stable fingerprints. Bigram shingling
// would be more discriminating but belongs with the language layer.
func Tokenize(s string) []string {
	s = strings.ToLower(s)

	var tokens []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		tok := cur.String()
		cur.Reset()
		if len(tok) < 3 {
			return
		}
		if _, ok := stopwords[tok]; ok {
			return
		}
		tokens = append(tokens, tok)
	}

	for _, r := range s {
		if isCJK(r) {
			flush()
			tokens = append(tokens, string(r))
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func isCJK(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs
		return true
	case r >= 0x3040 && r <= 0x30FF: // Hiragana + Katakana
		return true
	case r >= 0xAC00 && r <= 0xD7AF: // Hangul Syllables
		return true
	}
	return false
}

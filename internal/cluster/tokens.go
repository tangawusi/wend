// Package cluster groups articles into events. It implements the
// incremental clustering described in the product specification:
// articles arrive continuously, each is assigned to an existing event
// or starts a new one, and the assignment is a weighted score over
// five independent signals.
//
// This package is deliberately not a general-purpose clustering library.
// It solves one problem: given a new article and a set of candidate
// events from the recent past, find the event (if any) that the article
// is reporting on.
package cluster

import (
	"strings"
	"unicode"
)

// ModelVersion identifies the clustering implementation. Bump on any
// change to the scoring function or the assignment threshold; event
// rows carry the value so a re-cluster can be audited.
const ModelVersion = "cluster/v0.1.0"

// maxBodyTokens caps how much of an article body participates in the
// body-similarity signal. The lede carries the event signal; the tail
// carries colour, quotes, and background. 300 tokens is roughly the
// first four paragraphs.
const maxBodyTokens = 300

// minTokenRunes filters single- and two-character tokens, which are
// overwhelmingly noise in news prose.
const minTokenRunes = 3

// clusterStopwords is intentionally short. A longer list improves
// title similarity slightly and hurts body similarity by removing
// words that distinguish one event from another within a topic.
var clusterStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "that": {}, "with": {}, "this": {},
	"from": {}, "have": {}, "has": {}, "had": {}, "are": {}, "was": {},
	"were": {}, "will": {}, "would": {}, "been": {}, "its": {}, "his": {},
	"her": {}, "their": {}, "there": {}, "they": {}, "them": {}, "than": {},
	"then": {}, "when": {}, "where": {}, "which": {}, "while": {}, "who": {},
	"whom": {}, "whose": {}, "what": {}, "into": {}, "onto": {}, "over": {},
	"under": {}, "about": {}, "after": {}, "before": {}, "during": {},
	"between": {}, "through": {}, "against": {}, "without": {},
}

// TokenSet returns the distinct content tokens of s, capped at limit.
// Pass limit <= 0 for no cap.
func TokenSet(s string, limit int) map[string]struct{} {
	out := make(map[string]struct{}, 64)

	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		tok := strings.ToLower(cur.String())
		cur.Reset()
		if len([]rune(tok)) < minTokenRunes {
			return
		}
		if _, stop := clusterStopwords[tok]; stop {
			return
		}
		out[tok] = struct{}{}
	}

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
		if limit > 0 && len(out) >= limit {
			return out
		}
	}
	flush()
	return out
}

// Jaccard returns |a ∩ b| / |a ∪ b|. Returns 0 when both sets are empty.
func Jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

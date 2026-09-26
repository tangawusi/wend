package dedupe

import (
	"math/bits"
	"strconv"
	"strings"
)

// SimHash64 returns the 64-bit SimHash fingerprint of a token stream.
// Returns 0 for empty input, which the caller must treat as "no
// fingerprint" rather than as a valid hash.
func SimHash64(tokens []string) uint64 {
	if len(tokens) == 0 {
		return 0
	}

	var v [64]int
	for _, tok := range tokens {
		h := fnv1a64(tok)
		for i := 0; i < 64; i++ {
			if h&(uint64(1)<<i) != 0 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	var out uint64
	for i := 0; i < 64; i++ {
		if v[i] > 0 {
			out |= uint64(1) << i
		}
	}
	return out
}

// HammingDistance returns the number of differing bits between two
// fingerprints. For news articles, 0–3 is almost certainly the same
// story; 4–6 is likely the same story with edits; above 10 is unrelated.
func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// BitsString formats a uint64 as a 64-character binary string suitable
// for PostgreSQL's bit(64) type.
func BitsString(h uint64) string {
	s := strconv.FormatUint(h, 2)
	if len(s) < 64 {
		s = strings.Repeat("0", 64-len(s)) + s
	}
	return s
}

// fnv1a64 is the FNV-1a hash. Chosen because it distributes well over
// short strings, has no dependencies, and is fast enough that hashing
// a few thousand tokens is negligible against the crawl itself.
func fnv1a64(s string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}

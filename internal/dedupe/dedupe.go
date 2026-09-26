package dedupe

// Fingerprint computes the SimHash of an article body. Returns 0 for
// content too short to fingerprint reliably; callers must treat 0 as
// "unknown" and skip dedup for that article.
func Fingerprint(title, body string) uint64 {
	tokens := Tokenize(title + " " + body)
	if len(tokens) < 20 {
		// Short inputs produce unstable SimHashes; SimHash assumes a
		// reasonable token count for the majority vote to converge.
		return 0
	}
	return SimHash64(tokens)
}

// MaxDistance is the default Hamming threshold for near-duplicate
// matching. Chosen conservatively: a false positive here means two
// distinct events are treated as one, which is worse for the product
// than a false negative.
const MaxDistance = 6

// WindowDays bounds the near-duplicate search. Re-reporting of a wire
// story after a month is rare enough that the scan cost is not worth
// the marginal recall.
const WindowDays = 30

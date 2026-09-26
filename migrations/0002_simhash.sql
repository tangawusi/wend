-- 0002_simhash.sql — SimHash fingerprint for near-duplicate detection.
--
-- SimHash64 is stored as bit(64). PostgreSQL 14+ provides bit_count()
-- for popcount and # for bitstring XOR, so the Hamming distance between
-- two fingerprints is `bit_count(a # b)`.
--
-- No index can accelerate popcount. The scan is instead bounded by a
-- time window via the partial index below; older rows are unlikely to
-- be near-duplicates of newly-discovered articles.

ALTER TABLE articles ADD COLUMN simhash bit(64);

CREATE INDEX articles_simhash_time_idx
  ON articles (discovered_at DESC)
  WHERE simhash IS NOT NULL;

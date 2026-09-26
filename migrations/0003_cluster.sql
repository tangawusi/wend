-- 0003_cluster.sql — support efficient incremental clustering.
--
-- Articles are assigned to at most one event at a time. A clustered_at
-- timestamp makes the "find unclustered articles" query an indexed scan
-- rather than an anti-join against event_articles, which matters once
-- article counts reach the tens of thousands.
--
-- Re-clustering an article (manual correction, model change) clears
-- clustered_at; the article becomes visible to the next clustering pass.

ALTER TABLE articles ADD COLUMN clustered_at timestamptz;

CREATE INDEX articles_unclustered_idx
  ON articles (discovered_at)
  WHERE clustered_at IS NULL;

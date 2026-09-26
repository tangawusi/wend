-- 0001_init.sql — canonical schema for wend.press.
-- Designed against internal/domain/*.go. No BEGIN/COMMIT: the runner
-- wraps each migration file in a transaction.
--
-- Partitioning strategy: articles and trajectory_observations are the
-- candidates for declarative monthly partitioning once volume justifies.
-- Their primary keys are structured so adding the time column later is
-- a mechanical change.

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ─── Sources ─────────────────────────────────────────────────────────────
CREATE TABLE sources (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name             text NOT NULL,
  domain           citext NOT NULL UNIQUE,
  country          varchar(2) NOT NULL,
  region           text,
  language         text NOT NULL,
  type             text NOT NULL CHECK (type IN
                     ('wire','broadcast','newspaper','portal','government','ngo')),
  rss_url          text,
  sitemap_url      text,
  crawl_policy     text NOT NULL DEFAULT 'default',
  reliability      text,
  active           boolean NOT NULL DEFAULT true,
  update_freq_secs bigint NOT NULL DEFAULT 900,
  registered_at    timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sources_active_country_idx ON sources (active, country) WHERE active;
CREATE INDEX sources_language_idx       ON sources (language)         WHERE active;

-- ─── Crawl runs ──────────────────────────────────────────────────────────
CREATE TABLE crawl_runs (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id        uuid NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  mode             text NOT NULL CHECK (mode IN
                     ('rss','sitemap','article','category','incremental','backfill')),
  started_at       timestamptz NOT NULL DEFAULT now(),
  finished_at      timestamptz,
  status           text NOT NULL DEFAULT 'running'
                     CHECK (status IN ('running','ok','failed','partial')),
  urls_discovered  integer NOT NULL DEFAULT 0,
  urls_fetched     integer NOT NULL DEFAULT 0,
  articles_created integer NOT NULL DEFAULT 0,
  error            text
);

CREATE INDEX crawl_runs_source_started_idx ON crawl_runs (source_id, started_at DESC);
CREATE INDEX crawl_runs_failed_idx         ON crawl_runs (status) WHERE status <> 'ok';

-- ─── Crawl URL state ─────────────────────────────────────────────────────
CREATE TABLE crawl_urls (
  url               text PRIMARY KEY,
  source_id         uuid NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  first_discovered  timestamptz NOT NULL DEFAULT now(),
  last_crawled      timestamptz,
  http_status       integer,
  content_hash      text,
  crawl_duration_ms integer,
  error             text,
  retry_count       integer NOT NULL DEFAULT 0,
  article_id        uuid
);

CREATE INDEX crawl_urls_source_last_idx ON crawl_urls (source_id, last_crawled DESC NULLS LAST);
CREATE INDEX crawl_urls_retry_idx       ON crawl_urls (retry_count) WHERE retry_count > 0;

-- ─── Raw article archive (spec §6) ───────────────────────────────────────
-- Preserves what the crawler actually fetched. Extraction algorithms
-- will change; we should not have to rescrape the world when they do.
-- At very large scale this table moves to object storage; the shape
-- stays the same.
CREATE TABLE raw_articles (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id       uuid NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  url             text NOT NULL,
  fetched_at      timestamptz NOT NULL DEFAULT now(),
  http_status     integer NOT NULL,
  content_type    text,
  body            bytea NOT NULL,
  content_hash    text NOT NULL,
  parser_version  text NOT NULL,
  UNIQUE (url, content_hash)
);

CREATE INDEX raw_articles_source_fetched_idx ON raw_articles (source_id, fetched_at DESC);
CREATE INDEX raw_articles_hash_idx           ON raw_articles (content_hash);

-- ─── Articles ────────────────────────────────────────────────────────────
CREATE TABLE articles (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id           uuid NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
  url                 text NOT NULL,
  canonical_url       text NOT NULL,
  title               text NOT NULL,
  body                text NOT NULL,
  author              text,
  section             text,
  published_at        timestamptz,
  discovered_at       timestamptz NOT NULL DEFAULT now(),
  language            text NOT NULL,
  publication_country varchar(2) NOT NULL,
  content_hash        text NOT NULL,
  search_tsv          tsvector GENERATED ALWAYS AS (
                        setweight(to_tsvector('simple', coalesce(title,'')), 'A') ||
                        setweight(to_tsvector('simple', coalesce(body,'')),  'B')
                      ) STORED,
  parser_version      text NOT NULL,
  model_version       text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (source_id, url),
  UNIQUE (source_id, canonical_url)
);

CREATE INDEX articles_hash_idx             ON articles (content_hash);
CREATE INDEX articles_source_published_idx ON articles (source_id, published_at DESC NULLS LAST);
CREATE INDEX articles_language_pub_idx     ON articles (language, published_at DESC NULLS LAST);
CREATE INDEX articles_country_pub_idx      ON articles (publication_country, published_at DESC NULLS LAST);
CREATE INDEX articles_search_tsv_idx       ON articles USING gin (search_tsv);
CREATE INDEX articles_title_trgm_idx       ON articles USING gin (title gin_trgm_ops);

-- ─── Article translations (spec §8) ──────────────────────────────────────
-- Article.body is authoritative. Translations live here, one row per
-- target language, each with its own provenance.
CREATE TABLE article_translations (
  article_id     uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  to_language    text NOT NULL,
  from_language  text NOT NULL,
  body           text NOT NULL,
  model          text NOT NULL,
  model_version  text NOT NULL,
  translated_at  timestamptz NOT NULL DEFAULT now(),
  source_hash    text NOT NULL,
  output_hash    text NOT NULL,
  PRIMARY KEY (article_id, to_language)
);

-- ─── Article versions ────────────────────────────────────────────────────
-- If a source edits an article in place, prior text is preserved.
CREATE TABLE article_versions (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  article_id     uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  title          text NOT NULL,
  body           text NOT NULL,
  content_hash   text NOT NULL,
  captured_at    timestamptz NOT NULL DEFAULT now(),
  raw_article_id uuid REFERENCES raw_articles(id) ON DELETE SET NULL
);

CREATE INDEX article_versions_article_idx ON article_versions (article_id, captured_at DESC);

-- ─── Entities (spec §10) ─────────────────────────────────────────────────
CREATE TABLE entities (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kind        text NOT NULL CHECK (kind IN
                ('person','organization','country','city','location',
                 'government','institution','company','product','event')),
  canonical   text NOT NULL,
  aliases     text[] NOT NULL DEFAULT '{}',
  country     varchar(2),
  wikidata_id text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (kind, canonical)
);

CREATE INDEX entities_aliases_idx   ON entities USING gin (aliases);
CREATE INDEX entities_canonical_trgm ON entities USING gin (canonical gin_trgm_ops);

CREATE TABLE article_entities (
  article_id uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  entity_id  uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
  mentions   integer NOT NULL DEFAULT 1,
  salience   real NOT NULL DEFAULT 0,
  PRIMARY KEY (article_id, entity_id)
);

CREATE INDEX article_entities_entity_idx ON article_entities (entity_id);

-- ─── Locations ───────────────────────────────────────────────────────────
CREATE TABLE locations (
  id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  country varchar(2) NOT NULL,
  region  text,
  city    text,
  lat     double precision,
  lon     double precision,
  UNIQUE (country, region, city)
);

-- ─── Events (spec §11) ───────────────────────────────────────────────────
CREATE TABLE events (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title              text NOT NULL,
  summary            text,
  state              text NOT NULL CHECK (state IN
                       ('emerging','active','developing','cooling','resolved','stale')),
  start_time         timestamptz NOT NULL,
  last_observed_time timestamptz NOT NULL,
  article_count      integer NOT NULL DEFAULT 0,
  source_count       integer NOT NULL DEFAULT 0,
  language_count     integer NOT NULL DEFAULT 0,
  cluster_confidence real NOT NULL DEFAULT 0,
  model_version      text NOT NULL,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX events_state_observed_idx ON events (state, last_observed_time DESC);
CREATE INDEX events_start_idx          ON events (start_time DESC);
CREATE INDEX events_updated_idx        ON events (updated_at DESC);

CREATE TABLE event_articles (
  event_id   uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  article_id uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  similarity real NOT NULL DEFAULT 0,
  added_at   timestamptz NOT NULL DEFAULT now(),
  pinned     boolean NOT NULL DEFAULT false,
  excluded   boolean NOT NULL DEFAULT false,
  PRIMARY KEY (event_id, article_id)
);

CREATE INDEX event_articles_article_idx ON event_articles (article_id);

CREATE TABLE event_entities (
  event_id  uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  entity_id uuid NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
  role      text,
  PRIMARY KEY (event_id, entity_id)
);

CREATE TABLE event_locations (
  event_id    uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  role        text NOT NULL CHECK (role IN
                ('occurred','reported-from','affected','mentioned')),
  confidence  real NOT NULL DEFAULT 0,
  PRIMARY KEY (event_id, location_id, role)
);

-- ─── Trajectories (spec §14, §16, §17) ───────────────────────────────────
-- Append-only. Corrections are new rows. The audit story depends on
-- never mutating a trajectory that has been emitted.
CREATE TABLE trajectories (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id        uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  observed_at     timestamptz NOT NULL,
  direction       text NOT NULL CHECK (direction IN
                    ('escalating','stable','deescalating','unclear')),
  momentum        real NOT NULL,
  confidence      real NOT NULL,
  coverage_tier   text NOT NULL CHECK (coverage_tier IN
                    ('thin','moderate','high')),
  coverage        jsonb NOT NULL,
  confidence_expl jsonb NOT NULL,
  model_version   text NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, observed_at)
);

CREATE INDEX trajectories_event_observed_idx ON trajectories (event_id, observed_at DESC);

-- Raw input time series that the trajectory engine consumes.
CREATE TABLE trajectory_observations (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id       uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  observed_at    timestamptz NOT NULL,
  article_count  integer NOT NULL,
  source_count   integer NOT NULL,
  language_count integer NOT NULL,
  country_count  integer NOT NULL,
  UNIQUE (event_id, observed_at)
);

CREATE INDEX trajectory_obs_event_time_idx ON trajectory_observations (event_id, observed_at DESC);

-- ─── Claims & contradictions (spec §15) ──────────────────────────────────
CREATE TABLE claims (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  article_id   uuid NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  text         text NOT NULL,
  kind         text,
  value_num    double precision,
  value_text   text,
  confidence   real NOT NULL DEFAULT 0,
  extracted_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX claims_article_idx ON claims (article_id);

CREATE TABLE contradictions (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  claim_a_id    uuid NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
  claim_b_id    uuid NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
  kind          text NOT NULL,
  severity      real NOT NULL DEFAULT 0,
  resolved      boolean NOT NULL DEFAULT false,
  resolved_note text,
  detected_at   timestamptz NOT NULL DEFAULT now(),
  CHECK (claim_a_id < claim_b_id)
);

CREATE INDEX contradictions_event_idx ON contradictions (event_id) WHERE NOT resolved;

-- ─── Provenance & anchors (spec §18) ─────────────────────────────────────
CREATE TABLE provenance_records (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  object_type    text NOT NULL,
  object_id      uuid NOT NULL,
  action         text NOT NULL,
  content_hash   text NOT NULL,
  derived_from   uuid[] NOT NULL DEFAULT '{}',
  parser_version text,
  model_version  text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  anchor_id      uuid
);

CREATE INDEX prov_object_idx      ON provenance_records (object_type, object_id, created_at DESC);
CREATE INDEX prov_hash_idx        ON provenance_records (content_hash);
CREATE INDEX prov_unanchored_idx  ON provenance_records (created_at) WHERE anchor_id IS NULL;

CREATE TABLE anchors (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  merkle_root     text NOT NULL,
  record_count    integer NOT NULL,
  first_record_at timestamptz NOT NULL,
  last_record_at  timestamptz NOT NULL,
  chain           text,
  txid            text,
  block_height    bigint,
  anchored_at     timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX anchors_unanchored_idx ON anchors (created_at) WHERE anchored_at IS NULL;

ALTER TABLE provenance_records
  ADD CONSTRAINT prov_anchor_fk
  FOREIGN KEY (anchor_id) REFERENCES anchors(id) ON DELETE SET NULL;

package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"wend.press/internal/dedupe"
	"wend.press/internal/entity"
	"wend.press/internal/geo"
	"wend.press/internal/lang"
	"wend.press/internal/normalize"
	"wend.press/internal/store"
)

type Runner struct {
	store     *store.Store
	adapter   SourceAdapter
	cfg       CollectorConfig
	langSvc   *lang.Service
	extract   *entity.Extractor
	gazette   *geo.Gazetteer
	countries *geo.Registry
}

func NewRunner(
	s *store.Store,
	a SourceAdapter,
	cfg CollectorConfig,
	langSvc *lang.Service,
	extract *entity.Extractor,
	gazette *geo.Gazetteer,
	countries *geo.Registry,
) *Runner {
	return &Runner{
		store: s, adapter: a, cfg: cfg,
		langSvc: langSvc, extract: extract,
		gazette: gazette, countries: countries,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	src := r.adapter.Source()
	slog.Info("crawl starting", "source", src.Name, "rss", src.RSSURL)

	sourceID, err := r.store.UpsertSource(ctx, store.SourceInput{
		Name: src.Name, Domain: src.Domain,
		Country: string(src.Country), Region: src.Region,
		Language: string(src.Language), Type: string(src.Type),
		RSSURL: src.RSSURL, SitemapURL: src.SitemapURL,
	})
	if err != nil {
		return fmt.Errorf("upsert source: %w", err)
	}

	runID, err := r.store.StartCrawlRun(ctx, sourceID, "rss")
	if err != nil {
		return fmt.Errorf("start crawl run: %w", err)
	}

	items, err := r.adapter.Discover(ctx)
	if err != nil {
		_ = r.store.FinishCrawlRun(ctx, runID, "failed", 0, 0, 0, err.Error())
		return fmt.Errorf("discover: %w", err)
	}
	slog.Info("discovered", "count", len(items))

	opts := normalize.Options{SourceDomain: src.Domain, SourceName: src.Name}

	var created, failed, entitiesLinked int
	for _, it := range items {
		if err := ctx.Err(); err != nil {
			_ = r.store.FinishCrawlRun(ctx, runID, "partial", len(items), created, failed, "context cancelled")
			return err
		}
		n, err := r.ingestOne(ctx, sourceID, it, opts)
		if err != nil {
			failed++
			slog.Warn("ingest failed", "url", it.URL, "err", err)
			continue
		}
		created++
		entitiesLinked += n
	}

	status := "ok"
	switch {
	case failed > 0 && created > 0:
		status = "partial"
	case failed > 0 && created == 0:
		status = "failed"
	}
	_ = r.store.FinishCrawlRun(ctx, runID, status, len(items), created, failed, "")

	slog.Info("crawl finished",
		"source", src.Name, "discovered", len(items),
		"created", created, "failed", failed,
		"entities_linked", entitiesLinked, "status", status)
	return nil
}

func (r *Runner) ingestOne(ctx context.Context, sourceID uuid.UUID, it Discovered, opts normalize.Options) (int, error) {
	src := r.adapter.Source()

	ext, err := r.adapter.Extract(ctx, it.URL)
	if err != nil {
		return 0, fmt.Errorf("extract: %w", err)
	}

	langHint := string(src.Language)
	if langHint == "" {
		langHint = "und"
	}

	out := normalize.Article(normalize.Input{
		URL: it.URL, Title: it.Title, RawHTML: ext.RawHTML,
		FallbackBody: it.Summary,
		Author:       firstNonEmpty(it.Author, ext.Author),
		PublishedAt:  it.PublishedAt, Language: langHint,
	}, opts)

	if len([]rune(out.Body)) < 50 {
		return 0, fmt.Errorf("body too short after normalization")
	}

	lr := r.langSvc.Process(ctx, out.Title, out.Body, langHint)

	if len(ext.RawHTML) > 0 {
		if _, err := r.store.SaveRawArticle(ctx, store.RawArticleInput{
			SourceID: sourceID, URL: it.URL, HTTPStatus: 200,
			ContentType: "text/html", Body: ext.RawHTML,
			ContentHash:   hashString(string(ext.RawHTML)),
			ParserVersion: ParserVersion,
		}); err != nil {
			slog.Warn("save raw article", "url", it.URL, "err", err)
		}
	}

	articleID, err := r.store.SaveArticle(ctx, store.ArticleInput{
		SourceID: sourceID, URL: out.URL, CanonicalURL: out.CanonicalURL,
		Title: out.Title, Body: out.Body, Author: out.Author,
		PublishedAt: out.PublishedAt, Language: lr.Language,
		Country: string(src.Country), ContentHash: out.ContentHash,
		ParserVersion: ParserVersion, ModelVersion: normalize.Version,
	})
	if err != nil {
		return 0, fmt.Errorf("save article: %w", err)
	}

	if lr.Translation != nil {
		if err := r.store.SaveTranslation(ctx, store.TranslationInput{
			ArticleID: articleID,
			ToLanguage: lr.Translation.ToLanguage,
			FromLanguage: lr.Translation.FromLanguage,
			Body: lr.TranslatedBody, Model: lr.Translation.Model,
			ModelVersion: lr.Translation.ModelVersion,
			SourceHash: lr.Translation.SourceHash,
			OutputHash: lr.Translation.OutputHash,
		}); err != nil {
			slog.Warn("save translation", "article", articleID, "err", err)
		}
	}

	n := r.extractAndLink(ctx, articleID, out.Title, out.Body)
	r.fingerprintAndCheck(ctx, articleID, out.Title, out.Body)
	return n, nil
}

func (r *Runner) extractAndLink(ctx context.Context, articleID uuid.UUID, title, body string) int {
	ents := r.extract.Extract(title, body)
	if len(ents) == 0 {
		return 0
	}
	linked := 0
	for _, e := range ents {
		kind := string(e.Kind)
		country := ""
		if c, ok := r.countries.Resolve(e.Text); ok {
			country = c.Alpha2
		} else if p, ok := r.gazette.Resolve(e.Text, ""); ok {
			country = p.Alpha2
			if kind == string(entity.KindCountry) {
				kind = string(entity.KindCity)
			}
		}
		entityID, err := r.store.UpsertEntity(ctx, store.EntityInput{
			Kind: kind, Name: e.Text, Country: country,
		})
		if err != nil {
			slog.Warn("upsert entity", "name", e.Text, "err", err)
			continue
		}
		if err := r.store.LinkArticleEntity(ctx, store.ArticleEntityInput{
			ArticleID: articleID, EntityID: entityID,
			Mentions: e.Mentions, Salience: e.Salience,
		}); err != nil {
			slog.Warn("link entity", "name", e.Text, "err", err)
			continue
		}
		linked++
	}
	return linked
}

func (r *Runner) fingerprintAndCheck(ctx context.Context, articleID uuid.UUID, title, body string) {
	fp := dedupe.Fingerprint(title, body)
	if fp == 0 {
		return
	}
	bits := dedupe.BitsString(fp)
	if err := r.store.SetSimHash(ctx, articleID, bits); err != nil {
		slog.Warn("set simhash", "article", articleID, "err", err)
		return
	}
	dupes, err := r.store.FindNearDuplicates(ctx, bits, dedupe.MaxDistance, dedupe.WindowDays, articleID)
	if err != nil {
		slog.Warn("find near duplicates", "article", articleID, "err", err)
		return
	}
	for _, d := range dupes {
		slog.Info("near-duplicate", "distance", d.Distance, "source", d.SourceName, "title", truncate(d.Title, 60))
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wend.press/internal/cluster"
	"wend.press/internal/config"
	"wend.press/internal/entity"
	"wend.press/internal/geo"
	"wend.press/internal/ingest"
	"wend.press/internal/lang"
	"wend.press/internal/store"
	"wend.press/internal/web"
)

func main() {
	args := os.Args[1:]
	switch {
	case len(args) > 0 && args[0] == "migrate":
		fatal(runMigrate())
	case len(args) > 0 && args[0] == "crawl":
		source := "all"
		if len(args) > 1 {
			source = args[1]
		}
		fatal(runCrawl(source))
	case len(args) > 0 && args[0] == "cluster":
		fatal(runCluster())
	default:
		fatal(runServer())
	}
}

func fatal(err error) {
	if err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func setupLogging(cfg config.Config) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))
}

func runMigrate() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return store.Migrate(ctx, pool)
}

func runCluster() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	st := store.New(pool)
	clusterer := cluster.NewClusterer(st)

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	return clusterer.Run(runCtx)
}

func runCrawl(source string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	st := store.New(pool)
	ccfg := ingest.DefaultCollectorConfig()

	detector, err := lang.NewDetector(cfg.LangDetect)
	if err != nil {
		return fmt.Errorf("lang detector: %w", err)
	}
	langSvc := lang.NewService(detector, nil, cfg.LangTarget)
	slog.Info("language service ready",
		"target", cfg.LangTarget, "candidates", len(cfg.LangDetect))

	gazetteer := geo.NewGazetteer(cfg.GazetteerPath)
	if n := gazetteer.Size(); n > 0 {
		slog.Info("gazetteer loaded", "places", n, "path", cfg.GazetteerPath)
	} else {
		slog.Warn("gazetteer empty; city resolution disabled", "path", cfg.GazetteerPath)
	}
	countries := geo.NewRegistry()
	extractor := entity.NewExtractor()

	adapters, err := adaptersFor(source, ccfg)
	if err != nil {
		return err
	}

	crawlCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	var firstErr error
	for _, a := range adapters {
		if err := ctx.Err(); err != nil {
			return err
		}
		runner := ingest.NewRunner(st, a, ccfg, langSvc, extractor, gazetteer, countries)
		if err := runner.Run(crawlCtx); err != nil {
			slog.Error("crawl failed", "source", a.Source().Name, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func adaptersFor(name string, ccfg ingest.CollectorConfig) ([]ingest.SourceAdapter, error) {
	switch name {
	case "bbc":
		return []ingest.SourceAdapter{ingest.NewBBC(ccfg)}, nil
	case "guardian":
		return []ingest.SourceAdapter{ingest.NewGuardian(ccfg)}, nil
	case "all":
		return []ingest.SourceAdapter{
			ingest.NewBBC(ccfg),
			ingest.NewGuardian(ccfg),
		}, nil
	}
	return nil, fmt.Errorf("unknown source %q (try: bbc, guardian, all)", name)
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg)

	ui, err := web.New()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	ui.Routes(mux)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

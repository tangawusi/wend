package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	LogLevel    slog.Level
	DatabaseURL string

	LangTarget    string
	LangDetect    []string
	GazetteerPath string
}

const defaultDetectLanguages = "en,es,fr,de,pt,it,nl,ru,ar,zh,ja,ko,tr,pl,uk,fa,hi,id,vi,th"

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      envOr("WEND_HTTP_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("WEND_DATABASE_URL"),
		LangTarget:    envOr("WEND_LANG_TARGET", "en"),
		LangDetect:    splitCSV(envOr("WEND_LANG_DETECT", defaultDetectLanguages)),
		GazetteerPath: os.Getenv("WEND_GAZETTEER_PATH"),
	}

	level, err := parseLevel(envOr("WEND_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("WEND_DATABASE_URL is required")
	}
	return cfg, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("unknown log level %q", s)
}

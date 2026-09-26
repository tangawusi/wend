package config

import (
	"fmt"

	"github.com/joho/godotenv"
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
	// Best-effort: .env may not exist in production, where the
	// orchestrator populates the environment directly. A missing
	// file is not an error; a malformed one is logged and ignored.
	// Both paths checked: .env at cwd covers `make run` from the
	// repo root; configs/.env covers running the binary from bin/.
	if err := godotenv.Load(".env", "configs/.env"); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("dotenv load", "err", err)
		}
	}

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

package config

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	HTTPAddr    string
	LogLevel    slog.Level
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    envOr("WEND_HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("WEND_DATABASE_URL"),
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

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wend.press/migrations"
)

// advisoryLockKey serializes migrations across processes. Value is
// arbitrary but stable: 'wend' as a 32-bit int.
const advisoryLockKey int64 = 0x77656e64

// Migrate applies every pending migration in lexical order.
// Safe to run concurrently; takes a Postgres advisory lock.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.Background(),
			"SELECT pg_advisory_unlock($1)", advisoryLockKey)
	}()

	if err := ensureMigrationsTable(ctx, conn.Conn()); err != nil {
		return err
	}

	applied, err := appliedMigrations(ctx, conn.Conn())
	if err != nil {
		return err
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}

	for _, name := range files {
		if _, ok := applied[name]; ok {
			continue
		}
		if err := applyMigration(ctx, conn.Conn(), name); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		slog.Info("migration applied", "name", name)
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, c *pgx.Conn) error {
	_, err := c.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       text PRIMARY KEY,
			checksum   text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`)
	return err
}

func appliedMigrations(ctx context.Context, c *pgx.Conn) (map[string]struct{}, error) {
	rows, err := c.Query(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = struct{}{}
	}
	return out, rows.Err()
}

func migrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func applyMigration(ctx context.Context, c *pgx.Conn, name string) error {
	data, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op after successful Commit

	if _, err := tx.Exec(ctx, string(data)); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (name, checksum) VALUES ($1, $2)",
		name, checksum); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

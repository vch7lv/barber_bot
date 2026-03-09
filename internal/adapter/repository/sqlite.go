package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB открывает SQLite по DSN и применяет миграции.
func DB(ctx context.Context, dsn string, log *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("read migration %s: %w", e.Name(), err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("run migration %s: %w", e.Name(), err)
		}
		log.Info("database migrated", "file", e.Name())
	}
	return db, nil
}

package db

import (
	"context"
	"embed"
	"fmt"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate serializes concurrent upgrades and commits the schema and its ledger together.
func (s *Store) Migrate(ctx context.Context) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// The same lock is shared by every Claritty migrator in this database.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1129070932, 1)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS claritty_schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	for _, file := range files {
		version := file.Name()
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM claritty_schema_migrations WHERE version = $1
		)`, version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		statement, err := migrations.ReadFile("migrations/" + version)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO claritty_schema_migrations (version) VALUES ($1)`, version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CheckSchema(ctx context.Context) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	latest := files[len(files)-1].Name()
	var applied bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM claritty_schema_migrations WHERE version = $1
	)`, latest).Scan(&applied); err != nil {
		return err
	}
	if !applied {
		return fmt.Errorf("required migration %s is missing", latest)
	}
	return nil
}

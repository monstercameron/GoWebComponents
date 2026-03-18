package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Migrate(ctx context.Context, database *sql.DB, migrationsDir string, fallbackSchemaPath string) error {
	if _, err := database.ExecContext(ctx, `create table if not exists schema_migrations (version text primary key, applied_at text not null default current_timestamp)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	migrations, err := loadMigrations(migrationsDir, fallbackSchemaPath)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		var applied string
		scanErr := database.QueryRowContext(ctx, `select version from schema_migrations where version = ?`, migration.Version).Scan(&applied)
		if scanErr == nil {
			continue
		}
		if scanErr != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", migration.Version, scanErr)
		}
		if _, err := database.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		if _, err := database.ExecContext(ctx, `insert into schema_migrations(version) values (?)`, migration.Version); err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Version, err)
		}
	}
	return nil
}

type migration struct {
	Version string
	SQL     string
}

func loadMigrations(migrationsDir string, fallbackSchemaPath string) ([]migration, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err == nil {
		migrations := make([]migration, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			content, readErr := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
			if readErr != nil {
				return nil, fmt.Errorf("read migration %s: %w", entry.Name(), readErr)
			}
			migrations = append(migrations, migration{Version: entry.Name(), SQL: string(content)})
		}
		sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
		if len(migrations) > 0 {
			return migrations, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	content, err := os.ReadFile(fallbackSchemaPath)
	if err != nil {
		if err == fs.ErrNotExist || os.IsNotExist(err) {
			return nil, fmt.Errorf("no migrations and no fallback schema at %s", fallbackSchemaPath)
		}
		return nil, fmt.Errorf("read fallback schema: %w", err)
	}
	return []migration{{Version: "001_initial_schema.sql", SQL: string(content)}}, nil
}

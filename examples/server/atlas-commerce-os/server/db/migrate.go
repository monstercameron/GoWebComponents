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

func Migrate(parseCtx context.Context, parseDatabase *sql.DB, parseMigrationsDir string, parseFallbackSchemaPath string) error {
	if _, parseErr := parseDatabase.ExecContext(parseCtx, `create table if not exists schema_migrations (version text primary key, applied_at text not null default current_timestamp)`); parseErr != nil {
		return fmt.Errorf("create schema_migrations: %w", parseErr)
	}
	parseMigrations, parseErr2 := loadMigrations(parseMigrationsDir, parseFallbackSchemaPath)
	if parseErr2 != nil {
		return parseErr2
	}
	for _, parseMigration := range parseMigrations {
		var parseApplied string
		parseScanErr := parseDatabase.QueryRowContext(parseCtx, `select version from schema_migrations where version = ?`, parseMigration.Version).Scan(&parseApplied)
		if parseScanErr == nil {
			continue
		}
		if parseScanErr != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", parseMigration.Version, parseScanErr)
		}
		if _, parseErr3 := parseDatabase.ExecContext(parseCtx, parseMigration.SQL); parseErr3 != nil {
			return fmt.Errorf("apply migration %s: %w", parseMigration.Version, parseErr3)
		}
		if _, parseErr4 := parseDatabase.ExecContext(parseCtx, `insert into schema_migrations(version) values (?)`, parseMigration.Version); parseErr4 != nil {
			return fmt.Errorf("record migration %s: %w", parseMigration.Version, parseErr4)
		}
	}
	return nil
}

type migration struct {
	Version string
	SQL     string
}

func loadMigrations(parseMigrationsDir string, parseFallbackSchemaPath string) ([]migration, error) {
	parseEntries, parseErr := os.ReadDir(parseMigrationsDir)
	if parseErr == nil {
		parseMigrations := make([]migration, 0, len(parseEntries))
		for _, parseEntry := range parseEntries {
			if parseEntry.IsDir() || !strings.HasSuffix(parseEntry.Name(), ".sql") {
				continue
			}
			parseContent, parseReadErr := os.ReadFile(filepath.Join(parseMigrationsDir, parseEntry.Name()))
			if parseReadErr != nil {
				return nil, fmt.Errorf("read migration %s: %w", parseEntry.Name(), parseReadErr)
			}
			parseMigrations = append(parseMigrations, migration{Version: parseEntry.Name(), SQL: string(parseContent)})
		}
		sort.Slice(parseMigrations, func(parseI, parseJ int) bool {
			return parseMigrations[parseI].Version < parseMigrations[parseJ].Version
		})
		if len(parseMigrations) > 0 {
			return parseMigrations, nil
		}
	} else if !os.IsNotExist(parseErr) {
		return nil, fmt.Errorf("read migrations dir: %w", parseErr)
	}

	parseContent2, parseErr := os.ReadFile(parseFallbackSchemaPath)
	if parseErr != nil {
		if parseErr == fs.ErrNotExist || os.IsNotExist(parseErr) {
			return nil, fmt.Errorf("no migrations and no fallback schema at %s", parseFallbackSchemaPath)
		}
		return nil, fmt.Errorf("read fallback schema: %w", parseErr)
	}
	return []migration{{Version: "001_initial_schema.sql", SQL: string(parseContent2)}}, nil
}

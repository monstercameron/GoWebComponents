package main

import (
	"path/filepath"
	"testing"
)

// TestLoadConfigDatabaseOverride isolates test and developer databases without changing the default.
func TestLoadConfigDatabaseOverride(parseT *testing.T) {
	parsePath := filepath.Join(parseT.TempDir(), "isolated.db")
	parseT.Setenv("ATLAS_DB_PATH", "  "+parsePath+"  ")
	parseConfig, parseErr := loadConfig()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseConfig.SQLitePath != parsePath {
		parseT.Fatalf("database path=%q, want %q", parseConfig.SQLitePath, parsePath)
	}
	parseT.Setenv("ATLAS_DB_PATH", "")
	parseDefault, parseErr := loadConfig()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseWant := filepath.Join(parseDefault.ExampleRoot, "server", "data", "atlas-commerce-os.db")
	if parseDefault.SQLitePath != parseWant {
		parseT.Fatalf("default database path=%q, want %q", parseDefault.SQLitePath, parseWant)
	}
}

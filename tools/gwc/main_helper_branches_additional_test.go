package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestResolveBrowserWorkspaceCoversFallbacksAndErrors verifies override failures and workspace fallback discovery.
func TestResolveBrowserWorkspaceCoversFallbacksAndErrors(parseT *testing.T) {
	parseT.Run("rejects missing override directory", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"missing-browser"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		_, parseErr := resolveBrowserWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "configured browserWorkspace path does not exist") {
			parseT2.Fatalf("expected missing override error, got %v", parseErr)
		}
		if parseWorkspace := detectBrowserWorkspace(filepath.Join(parseRoot, "repo"), parseRoot); parseWorkspace != "" {
			parseT2.Fatalf("expected detectBrowserWorkspace to hide override errors, got %q", parseWorkspace)
		}
	})

	parseT.Run("rejects override without playwright suite", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseOverrideDir := filepath.Join(parseRoot, "enterprise-browser")
		if parseErr := os.MkdirAll(parseOverrideDir, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir override dir: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"browserWorkspace":"enterprise-browser"}}`), 0644); parseErr2 != nil {
			parseT2.Fatalf("write config: %v", parseErr2)
		}

		_, parseErr := resolveBrowserWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "does not contain a Playwright-Go suite") {
			parseT2.Fatalf("expected playwright suite error, got %v", parseErr)
		}
	})

	parseT.Run("prefers root playwright workspace before repo fallback", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.MkdirAll(filepath.Join(parseRoot, "playwrightgo"), 0755); parseErr != nil {
			parseT2.Fatalf("mkdir root playwright suite: %v", parseErr)
		}

		parseWorkspace, parseErr := resolveBrowserWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
		if parseErr != nil {
			parseT2.Fatalf("resolve browser workspace: %v", parseErr)
		}
		if parseWorkspace != parseRoot {
			parseT2.Fatalf("expected root browser workspace %q, got %q", parseRoot, parseWorkspace)
		}
	})

	parseT.Run("falls back to repo test workspace", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseRepoRoot := filepath.Join(parseRoot, "repo")
		parseRepoWorkspace := filepath.Join(parseRepoRoot, "test", "playwrightgo")
		if parseErr := os.MkdirAll(parseRepoWorkspace, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir repo playwright suite: %v", parseErr)
		}

		parseWorkspace, parseErr := resolveBrowserWorkspace(parseRepoRoot, parseRoot)
		if parseErr != nil {
			parseT2.Fatalf("resolve browser workspace: %v", parseErr)
		}
		if parseWorkspace != filepath.Join(parseRepoRoot, "test") {
			parseT2.Fatalf("expected repo fallback workspace %q, got %q", filepath.Join(parseRepoRoot, "test"), parseWorkspace)
		}
	})
}

// TestResolveLauncherLivereloadHelpersCoverDefaultsAndErrors verifies default resolution and invalid override handling.
func TestResolveLauncherLivereloadHelpersCoverDefaultsAndErrors(parseT *testing.T) {
	parseT.Run("returns default workspace without override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseRepoRoot := filepath.Join(parseRoot, "repo")

		parseWorkspace, parseErr := resolveLauncherLivereloadWorkspace(parseRepoRoot, parseRoot)
		if parseErr != nil {
			parseT2.Fatalf("resolve livereload workspace: %v", parseErr)
		}
		if parseWorkspace != filepath.Join(parseRepoRoot, "tools", "livereload") {
			parseT2.Fatalf("expected default workspace %q, got %q", filepath.Join(parseRepoRoot, "tools", "livereload"), parseWorkspace)
		}
	})

	parseT.Run("rejects missing workspace override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"livereloadWorkspace":"missing-livereload"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		_, parseErr := resolveLauncherLivereloadWorkspace(filepath.Join(parseRoot, "repo"), parseRoot)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "configured livereloadWorkspace path does not exist") {
			parseT2.Fatalf("expected missing livereload workspace error, got %v", parseErr)
		}
	})

	parseT.Run("returns no client script override when unset", func(parseT2 *testing.T) {
		parsePath, isParseOverride, parseErr := resolveLauncherLivereloadClientScript(parseT2.TempDir())
		if parseErr != nil {
			parseT2.Fatalf("resolve livereload client script: %v", parseErr)
		}
		if parsePath != "" || isParseOverride {
			parseT2.Fatalf("expected unset client script override, got path=%q override=%t", parsePath, isParseOverride)
		}
	})

	parseT.Run("rejects missing client script override", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"livereloadClientScript":"vendor/missing-client.js"}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write config: %v", parseErr)
		}

		_, _, parseErr := resolveLauncherLivereloadClientScript(parseRoot)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "configured livereloadClientScript path does not exist") {
			parseT2.Fatalf("expected missing client script override error, got %v", parseErr)
		}
	})
}

// TestSeedHelpersCoverNormalizationAndEnvReplacement verifies cwd failure, file-to-dir normalization, and env replacement rules.
func TestSeedHelpersCoverNormalizationAndEnvReplacement(parseT *testing.T) {
	parseT.Run("returns getwd failure", func(parseT2 *testing.T) {
		parseOriginalGetwd := seedGetwd
		parseT2.Cleanup(func() { seedGetwd = parseOriginalGetwd })
		seedGetwd = func() (string, error) {
			return "", errors.New("cwd failed")
		}

		_, parseErr := resolveSeedConfig(seedConfig{})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "cwd failed") {
			parseT2.Fatalf("expected getwd error, got %v", parseErr)
		}
	})

	parseT.Run("normalizes file command path and explicit db path", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseCommandDir := filepath.Join(parseRoot, "cmd", "seed")
		parseMainPath := filepath.Join(parseCommandDir, "main.go")
		if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir command dir: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
			parseT2.Fatalf("write main.go: %v", parseErr2)
		}

		parseOriginalGetwd := seedGetwd
		parseT2.Cleanup(func() { seedGetwd = parseOriginalGetwd })
		seedGetwd = func() (string, error) {
			return parseRoot, nil
		}

		parseConfig, parseErr := resolveSeedConfig(seedConfig{
			rootPath:    ".",
			commandPath: filepath.Join("cmd", "seed", "main.go"),
			dbPath:      filepath.Join("fixtures", "dev.db"),
		})
		if parseErr != nil {
			parseT2.Fatalf("resolve seed config: %v", parseErr)
		}
		if parseConfig.rootPath != parseRoot {
			parseT2.Fatalf("expected normalized root path %q, got %#v", parseRoot, parseConfig)
		}
		if parseConfig.commandPath != parseCommandDir {
			parseT2.Fatalf("expected normalized command dir %q, got %#v", parseCommandDir, parseConfig)
		}
		if parseConfig.dbPath != filepath.Join(parseRoot, "fixtures", "dev.db") {
			parseT2.Fatalf("expected normalized db path %q, got %#v", filepath.Join(parseRoot, "fixtures", "dev.db"), parseConfig)
		}
	})

	parseT.Run("returns empty default db for non chat wizard command", func(parseT2 *testing.T) {
		if parseGot := defaultSeedDatabasePath(filepath.Join(parseT2.TempDir(), "cmd", "seed")); parseGot != "" {
			parseT2.Fatalf("expected empty default db path for non chat wizard seeder, got %q", parseGot)
		}
	})

	parseT.Run("replaces only the first matching env entry and preserves order", func(parseT2 *testing.T) {
		parseEnv := []string{
			"PATH=/bin",
			"CHAT_DB_PATH=old-first.db",
			"USER=cam",
			"CHAT_DB_PATH=old-second.db",
		}

		parseUpdated := replaceEnvVar(parseEnv, "CHAT_DB_PATH", "new.db")
		parseWant := []string{
			"PATH=/bin",
			"CHAT_DB_PATH=new.db",
			"USER=cam",
		}
		if !slices.Equal(parseUpdated, parseWant) {
			parseT2.Fatalf("expected updated env %#v, got %#v", parseWant, parseUpdated)
		}

		parseAppended := replaceEnvVar([]string{"PATH=/bin"}, "CHAT_DB_PATH", "new.db")
		if !slices.Equal(parseAppended, []string{"PATH=/bin", "CHAT_DB_PATH=new.db"}) {
			parseT2.Fatalf("expected appended env entry, got %#v", parseAppended)
		}
	})
}

package sqlfiles

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func resetSQLFileState() {
	rootOnce = sync.Once{}
	roots = nil
	rootErr = nil
	cache = sync.Map{}
}

func TestLoadReadsAndCachesSQLFiles(parseT *testing.T) {
	resetSQLFileState()
	parseT.Cleanup(resetSQLFileState)

	parseRoot := parseT.TempDir()
	parseSqlDir := filepath.Join(parseRoot, "sql", "store")
	if parseErr := os.MkdirAll(parseSqlDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(): %v", parseErr)
	}
	parseFilePath := filepath.Join(parseSqlDir, "query.sql")
	if parseErr2 := os.WriteFile(parseFilePath, []byte("select 1;"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile(): %v", parseErr2)
	}

	if parseErr3 := os.Setenv("CHAT_WIZARD_ROOT", parseRoot); parseErr3 != nil {
		parseT.Fatalf("Setenv(): %v", parseErr3)
	}
	parseT.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	parseFirst, parseErr4 := ParseLoad("store/query.sql")
	if parseErr4 != nil {
		parseT.Fatalf("Load(): %v", parseErr4)
	}
	if parseFirst != "select 1;" {
		parseT.Fatalf("Load() = %q, want select 1;", parseFirst)
	}

	if parseErr5 := os.WriteFile(parseFilePath, []byte("select 2;"), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile(update): %v", parseErr5)
	}
	parseSecond, parseErr4 := ParseLoad("store/query.sql")
	if parseErr4 != nil {
		parseT.Fatalf("Load(cached): %v", parseErr4)
	}
	if parseSecond != parseFirst {
		parseT.Fatalf("expected cached SQL text, got %q want %q", parseSecond, parseFirst)
	}
}

func TestLoadRejectsEmptyAndMissingPaths(parseT *testing.T) {
	resetSQLFileState()
	parseT.Cleanup(resetSQLFileState)

	if _, parseErr := ParseLoad("   "); parseErr == nil || !strings.Contains(parseErr.Error(), "sql path is empty") {
		parseT.Fatalf("expected empty path error, got %v", parseErr)
	}

	parseRoot := parseT.TempDir()
	if parseErr2 := os.Setenv("CHAT_WIZARD_ROOT", parseRoot); parseErr2 != nil {
		parseT.Fatalf("Setenv(): %v", parseErr2)
	}
	parseT.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	_, parseErr3 := ParseLoad("store/missing.sql")
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "sql file not found") || !strings.Contains(parseErr3.Error(), "missing.sql") {
		parseT.Fatalf("expected missing file error, got %v", parseErr3)
	}
}

func TestExampleRootsDeduplicatesConfiguredAndDerivedPaths(parseT *testing.T) {
	resetSQLFileState()
	parseT.Cleanup(resetSQLFileState)

	parseRoot := parseT.TempDir()
	if parseErr := os.Setenv("CHAT_WIZARD_ROOT", parseRoot); parseErr != nil {
		parseT.Fatalf("Setenv(): %v", parseErr)
	}
	parseT.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	parseGot, parseErr2 := parseExampleRoots()
	if parseErr2 != nil {
		parseT.Fatalf("exampleRoots(): %v", parseErr2)
	}
	if len(parseGot) == 0 {
		parseT.Fatal("expected at least one example root")
	}
	parseSeen := map[string]struct{}{}
	for _, parseCurrent := range parseGot {
		if _, parseExists := parseSeen[parseCurrent]; parseExists {
			parseT.Fatalf("exampleRoots() returned duplicate root %q in %v", parseCurrent, parseGot)
		}
		parseSeen[parseCurrent] = struct{}{}
	}
	if _, parseOk := parseSeen[filepath.Clean(parseRoot)]; !parseOk {
		parseT.Fatalf("expected configured root %q in %v", parseRoot, parseGot)
	}
}

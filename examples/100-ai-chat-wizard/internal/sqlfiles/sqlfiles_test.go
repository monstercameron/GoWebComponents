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

func TestLoadReadsAndCachesSQLFiles(t *testing.T) {
	resetSQLFileState()
	t.Cleanup(resetSQLFileState)

	root := t.TempDir()
	sqlDir := filepath.Join(root, "sql", "store")
	if err := os.MkdirAll(sqlDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}
	filePath := filepath.Join(sqlDir, "query.sql")
	if err := os.WriteFile(filePath, []byte("select 1;"), 0o644); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if err := os.Setenv("CHAT_WIZARD_ROOT", root); err != nil {
		t.Fatalf("Setenv(): %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	first, err := Load("store/query.sql")
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if first != "select 1;" {
		t.Fatalf("Load() = %q, want select 1;", first)
	}

	if err := os.WriteFile(filePath, []byte("select 2;"), 0o644); err != nil {
		t.Fatalf("WriteFile(update): %v", err)
	}
	second, err := Load("store/query.sql")
	if err != nil {
		t.Fatalf("Load(cached): %v", err)
	}
	if second != first {
		t.Fatalf("expected cached SQL text, got %q want %q", second, first)
	}
}

func TestLoadRejectsEmptyAndMissingPaths(t *testing.T) {
	resetSQLFileState()
	t.Cleanup(resetSQLFileState)

	if _, err := Load("   "); err == nil || !strings.Contains(err.Error(), "sql path is empty") {
		t.Fatalf("expected empty path error, got %v", err)
	}

	root := t.TempDir()
	if err := os.Setenv("CHAT_WIZARD_ROOT", root); err != nil {
		t.Fatalf("Setenv(): %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	_, err := Load("store/missing.sql")
	if err == nil || !strings.Contains(err.Error(), "sql file not found") || !strings.Contains(err.Error(), "missing.sql") {
		t.Fatalf("expected missing file error, got %v", err)
	}
}

func TestExampleRootsDeduplicatesConfiguredAndDerivedPaths(t *testing.T) {
	resetSQLFileState()
	t.Cleanup(resetSQLFileState)

	root := t.TempDir()
	if err := os.Setenv("CHAT_WIZARD_ROOT", root); err != nil {
		t.Fatalf("Setenv(): %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("CHAT_WIZARD_ROOT") })

	got, err := exampleRoots()
	if err != nil {
		t.Fatalf("exampleRoots(): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one example root")
	}
	seen := map[string]struct{}{}
	for _, current := range got {
		if _, exists := seen[current]; exists {
			t.Fatalf("exampleRoots() returned duplicate root %q in %v", current, got)
		}
		seen[current] = struct{}{}
	}
	if _, ok := seen[filepath.Clean(root)]; !ok {
		t.Fatalf("expected configured root %q in %v", root, got)
	}
}

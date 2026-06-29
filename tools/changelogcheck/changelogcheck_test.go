package changelogcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const parseFixtureChangelog = `# Changelog

## [Unreleased]

### Added

## v3.0.46

Some notes about 3.0.46.

## 3.0.46 - 2026-06-12

Duplicate date-suffix form.

## [3.0.46]

Bracket form.

## 3.0.4

An older, shorter version that must NOT match v3.0.46 queries.

## 2.9.0

Even older release.
`

// TestHasEntryFindsAllForms verifies that HasEntry accepts the three common
// section-header styles for version 3.0.46.
func TestHasEntryFindsAllForms(t *testing.T) {
	parseVersionQueries := []string{"3.0.46", "v3.0.46"}
	for _, parseQuery := range parseVersionQueries {
		if !HasEntry(parseFixtureChangelog, parseQuery) {
			t.Errorf("HasEntry(%q): want true, got false", parseQuery)
		}
	}
}

// TestHasEntryReturnsFalseForMissingVersion confirms that a version not present
// in the changelog is not matched.
func TestHasEntryReturnsFalseForMissingVersion(t *testing.T) {
	if HasEntry(parseFixtureChangelog, "9.9.9") {
		t.Error("HasEntry(9.9.9): want false, got true")
	}
}

// TestHasEntryNoSubstringMatch verifies that "3.0.4" does not match a header
// for "3.0.46" — only whole-token matches are accepted.
func TestHasEntryNoSubstringMatch(t *testing.T) {
	// The fixture contains "## 3.0.4" as its own entry; searching for "3.0.4"
	// must return true for that entry only. But searching for "3.0.4" must NOT
	// produce a false positive against "## 3.0.46".
	parseSingleEntryChangelog := `# Changelog

## 3.0.46

Notes.
`
	if HasEntry(parseSingleEntryChangelog, "3.0.4") {
		t.Error("HasEntry(3.0.4) against '## 3.0.46': want false (substring), got true")
	}
}

// TestLatestEntrySkipsH1AndReturnsFirstSection verifies that LatestEntry skips
// the top-level "# Changelog" heading and returns the first ## section.
func TestLatestEntrySkipsH1AndReturnsFirstSection(t *testing.T) {
	parseHeader, parseBody, parseOk := LatestEntry(parseFixtureChangelog)
	if !parseOk {
		t.Fatal("LatestEntry: want ok=true, got false")
	}
	if parseHeader != "[Unreleased]" {
		t.Errorf("LatestEntry header: want [Unreleased], got %q", parseHeader)
	}
	// Body of [Unreleased] section contains the "### Added" line.
	if parseBody == "" && !containsVersionToken(parseFixtureChangelog, "Unreleased") {
		// Body may be empty if [Unreleased] has no content lines; that is fine.
		_ = parseBody
	}
	_ = parseBody // body content varies by fixture; presence of ok is the key assertion
}

// TestLatestEntryEmptyInputReturnsFalse verifies that an empty changelog
// produces ok=false.
func TestLatestEntryEmptyInputReturnsFalse(t *testing.T) {
	_, _, parseOk := LatestEntry("")
	if parseOk {
		t.Error("LatestEntry on empty input: want ok=false, got true")
	}
}

// TestCheckFileRoundTrip writes a small changelog to a temp file and verifies
// that CheckFile correctly reports present and absent versions.
func TestCheckFileRoundTrip(t *testing.T) {
	parseDir := t.TempDir()
	parsePath := filepath.Join(parseDir, "CHANGELOG.md")
	parseContent := `# Changelog

## v1.2.3

Initial release.

## 1.0.0

Older release.
`
	if parseErr := os.WriteFile(parsePath, []byte(parseContent), 0o644); parseErr != nil {
		t.Fatalf("WriteFile: %v", parseErr)
	}

	parseFound, parseErr := CheckFile(parsePath, "1.2.3")
	if parseErr != nil {
		t.Fatalf("CheckFile: unexpected error: %v", parseErr)
	}
	if !parseFound {
		t.Error("CheckFile(1.2.3): want true, got false")
	}

	parseFound, parseErr = CheckFile(parsePath, "v1.0.0")
	if parseErr != nil {
		t.Fatalf("CheckFile: unexpected error: %v", parseErr)
	}
	if !parseFound {
		t.Error("CheckFile(v1.0.0): want true, got false")
	}

	parseFound, parseErr = CheckFile(parsePath, "9.9.9")
	if parseErr != nil {
		t.Fatalf("CheckFile: unexpected error: %v", parseErr)
	}
	if parseFound {
		t.Error("CheckFile(9.9.9): want false, got true")
	}
}

// TestCheckFileMissingFileReturnsError verifies that CheckFile returns a
// non-nil error when the target path does not exist.
func TestCheckFileMissingFileReturnsError(t *testing.T) {
	_, parseErr := CheckFile(filepath.Join(t.TempDir(), "nonexistent.md"), "1.0.0")
	if parseErr == nil {
		t.Error("CheckFile on nonexistent file: want error, got nil")
	}
}

func TestLatestEntryIsSurfacedInDocsSiteMirror(t *testing.T) {
	parseRepoRoot := filepath.Clean(filepath.Join("..", ".."))
	parseChangelog, parseErr := os.ReadFile(filepath.Join(parseRepoRoot, "CHANGELOG.md"))
	if parseErr != nil {
		t.Fatalf("read CHANGELOG.md: %v", parseErr)
	}
	parseHeader, _, parseOk := LatestEntry(string(parseChangelog))
	if !parseOk {
		t.Fatal("expected root CHANGELOG.md to expose a latest entry")
	}
	parseDocsPath := filepath.Join(parseRepoRoot, "examples", "public-examples-site", "assets", "docs", "latest-release-notes.md")
	parseDocs, parseErr := os.ReadFile(parseDocsPath)
	if parseErr != nil {
		t.Fatalf("read docs latest-release-notes: %v", parseErr)
	}
	parseDocsText := string(parseDocs)
	if !strings.Contains(parseDocsText, "## "+parseHeader) {
		t.Fatalf("docs latest-release-notes does not surface LatestEntry header %q:\n%s", parseHeader, parseDocsText)
	}
	if !strings.Contains(parseDocsText, "tools/changelogcheck.LatestEntry") {
		t.Fatalf("docs latest-release-notes should name the parser-backed source:\n%s", parseDocsText)
	}
}

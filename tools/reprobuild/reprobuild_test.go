package reprobuild

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up to the module root.
func repoRoot(parseT *testing.T) string {
	parseT.Helper()
	parseDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("getwd: %v", parseErr)
	}
	for range 8 {
		if _, parseStatErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseStatErr == nil {
			return parseDir
		}
		parseDir = filepath.Dir(parseDir)
	}
	parseT.Fatal("module root not found")
	return ""
}

// TestReproducibleWasmBuild builds the counter example to js/wasm twice in
// separate clean dirs and asserts the artifacts are byte-identical. This is the
// guard the provenance attestation needs: if a future change introduces
// nondeterminism (embedded timestamp, map iteration order in codegen, an
// untrimmed path), the two SHA-256 digests diverge and this fails.
func TestReproducibleWasmBuild(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("reproducible wasm build is slow; skipped in -short mode")
	}
	parseRoot := repoRoot(parseT)
	parsePackage := "./examples/public/counter"

	parseSHA1, parseErr1 := BuildWasmSHA(parseRoot, parsePackage, parseT.TempDir())
	if parseErr1 != nil {
		parseT.Fatalf("first build: %v", parseErr1)
	}
	parseSHA2, parseErr2 := BuildWasmSHA(parseRoot, parsePackage, parseT.TempDir())
	if parseErr2 != nil {
		parseT.Fatalf("second build: %v", parseErr2)
	}
	if !SHAMatches(parseSHA1, parseSHA2) {
		parseT.Fatalf("wasm build is not reproducible:\n  build 1: %s\n  build 2: %s", parseSHA1, parseSHA2)
	}
	parseT.Logf("reproducible: both builds = %s", parseSHA1)
}

// TestSHAMatchesCatchesDifference is the self-test proving the comparison
// detects nondeterminism: a single differing digit (as an embedded-timestamp
// fixture would produce) is reported as a mismatch, and the empty digest is
// never treated as a match.
func TestSHAMatchesCatchesDifference(parseT *testing.T) {
	if !SHAMatches("abc123", "abc123") {
		parseT.Fatal("identical digests should match")
	}
	if SHAMatches("abc123", "abc124") {
		parseT.Fatal("a one-character difference must be caught (nondeterminism)")
	}
	if SHAMatches("", "") {
		parseT.Fatal("empty digests must not count as a match")
	}
}

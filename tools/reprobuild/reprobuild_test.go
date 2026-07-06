package reprobuild

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if SHAMatches("abc123", "") || SHAMatches("", "abc123") {
		parseT.Fatal("one empty digest must not count as a match")
	}
	if SHAMatches("ABC123", "abc123") {
		parseT.Fatal("digest comparison must be exact and case-sensitive")
	}
}

func TestBuildWasmSHAIsDeterministicForTinyModule(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseT.Setenv("GOWORK", "off")

	writeReproFixture(parseT, parseRoot, "go.mod", `module example.com/reprofixture

go 1.21
`)
	writeReproFixture(parseT, parseRoot, "tiny/tiny.go", `package tiny

func Answer() int { return 42 }
`)

	parseDir1 := parseT.TempDir()
	parseSHA1, parseErr1 := BuildWasmSHA(parseRoot, "./tiny", parseDir1)
	if parseErr1 != nil {
		parseT.Fatalf("first tiny build: %v", parseErr1)
	}
	parseDir2 := parseT.TempDir()
	parseSHA2, parseErr2 := BuildWasmSHA(parseRoot, "./tiny", parseDir2)
	if parseErr2 != nil {
		parseT.Fatalf("second tiny build: %v", parseErr2)
	}

	if !SHAMatches(parseSHA1, parseSHA2) {
		parseT.Fatalf("tiny wasm build is not deterministic:\n  build 1: %s\n  build 2: %s", parseSHA1, parseSHA2)
	}
	if len(parseSHA1) != sha256.Size*2 {
		parseT.Fatalf("digest length = %d, want %d: %q", len(parseSHA1), sha256.Size*2, parseSHA1)
	}
	if parseSHA1 != strings.ToLower(parseSHA1) {
		parseT.Fatalf("digest must be lowercase hex: %q", parseSHA1)
	}

	parseArtifact, parseErr := os.ReadFile(filepath.Join(parseDir1, "repro.wasm"))
	if parseErr != nil {
		parseT.Fatalf("read built artifact: %v", parseErr)
	}
	parseManualSum := sha256.Sum256(parseArtifact)
	parseManualSHA := hex.EncodeToString(parseManualSum[:])
	if parseSHA1 != parseManualSHA {
		parseT.Fatalf("BuildWasmSHA returned %s, manual digest is %s", parseSHA1, parseManualSHA)
	}
}

func TestBuildWasmSHAReportsBuildFailure(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseT.Setenv("GOWORK", "off")

	writeReproFixture(parseT, parseRoot, "go.mod", `module example.com/reprofixture

go 1.21
`)

	_, parseErr := BuildWasmSHA(parseRoot, "./missing", parseT.TempDir())
	if parseErr == nil {
		parseT.Fatal("expected missing package build to fail")
	}
	parseMessage := parseErr.Error()
	if !strings.Contains(parseMessage, "build ./missing:") {
		parseT.Fatalf("error should include package context, got: %v", parseErr)
	}
	if !strings.Contains(parseMessage, "stat") && !strings.Contains(parseMessage, "cannot find") {
		parseT.Fatalf("error should include go build output, got: %v", parseErr)
	}
}

// TestReleaseWasmBuildArgsMatchReleaseProfile pins that the reproducibility gate
// builds with the SAME flags as the actual release profile (tools/gwc
// release_build.go "release": -trimpath, -ldflags "-s -w", -buildvcs=false,
// -tags production). Missing -tags production (dev vs production code) or
// -buildvcs=false (VCS-stamp nondeterminism the release strips) previously made
// the gate verify an artifact that is not what ships — a silent false-pass.
func TestReleaseWasmBuildArgsMatchReleaseProfile(parseT *testing.T) {
	parseArgs := releaseWasmBuildArgs("/tmp/out.wasm", "./pkg")
	parseJoined := strings.Join(parseArgs, " ")

	for _, parseNeed := range []string{"-trimpath", "-buildvcs=false"} {
		if !slices.Contains(parseArgs, parseNeed) {
			parseT.Fatalf("release build args missing %q: %v", parseNeed, parseArgs)
		}
	}
	if !strings.Contains(parseJoined, "-ldflags -s -w") {
		parseT.Fatalf("release build args missing ldflags -s -w: %v", parseArgs)
	}
	if !strings.Contains(parseJoined, "-tags production") {
		parseT.Fatalf("release build args missing -tags production (would verify the dev artifact, not the release): %v", parseArgs)
	}
	// Output + package must be the final positional args.
	if parseArgs[len(parseArgs)-2] != "/tmp/out.wasm" || parseArgs[len(parseArgs)-1] != "./pkg" {
		parseT.Fatalf("expected output + package as trailing args, got %v", parseArgs)
	}
}

func writeReproFixture(parseT *testing.T, parseRoot string, parseRelativePath string, parseContent string) {
	parseT.Helper()
	parsePath := filepath.Join(parseRoot, filepath.FromSlash(parseRelativePath))
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0755); parseErr != nil {
		parseT.Fatalf("create fixture directory: %v", parseErr)
	}
	if parseErr := os.WriteFile(parsePath, []byte(parseContent), 0644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}
}

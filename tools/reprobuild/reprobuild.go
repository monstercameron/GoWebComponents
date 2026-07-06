// Package reprobuild verifies that the release wasm build is reproducible: two
// independent builds of the same commit must produce byte-identical artifacts
// (the provenance attestation implicitly promises this). The `-trimpath` flag
// the release profile sets is what makes the output path-independent and thus
// deterministic.
package reprobuild

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildWasmSHA builds parsePackage for js/wasm with the RELEASE profile flags and
// returns the lowercase hex SHA-256 of the resulting artifact. The flags MUST
// match the actual release build (tools/gwc release_build.go "release" profile:
// Trimpath, Ldflags "-s -w", BuildVCS "false", Tags "production") — otherwise this
// gate verifies the reproducibility of an artifact that is NOT what ships. In
// particular -tags production selects the production code paths (a different
// binary than the default/dev build) and -buildvcs=false strips VCS stamping (a
// nondeterminism source the release deliberately removes); omitting either meant
// the check passed on the wrong artifact.
func BuildWasmSHA(parseRepoRoot string, parsePackage string, parseTempDir string) (string, error) {
	parseOut := filepath.Join(parseTempDir, "repro.wasm")
	parseCmd := exec.Command("go", releaseWasmBuildArgs(parseOut, parsePackage)...)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOutput, parseErr := parseCmd.CombinedOutput(); parseErr != nil {
		return "", fmt.Errorf("build %s: %w\n%s", parsePackage, parseErr, parseOutput)
	}
	parseData, parseErr := os.ReadFile(parseOut)
	if parseErr != nil {
		return "", parseErr
	}
	parseSum := sha256.Sum256(parseData)
	return hex.EncodeToString(parseSum[:]), nil
}

// releaseWasmBuildArgs is the `go build` argument list that reproduces the release
// profile. Kept as its own function so a test can pin that the release-matching
// flags (notably -tags production and -buildvcs=false) are present — without them
// the gate silently verifies the wrong artifact. Must stay in sync with the
// "release" profile in tools/gwc release_build.go.
func releaseWasmBuildArgs(parseOut string, parsePackage string) []string {
	return []string{
		"build",
		"-trimpath",
		"-ldflags", "-s -w",
		"-buildvcs=false",
		"-tags", "production",
		"-o", parseOut, parsePackage,
	}
}

// SHAMatches reports whether two artifact digests are identical. It is the
// comparison the reproducibility check uses; a single differing byte (for
// example an embedded build timestamp) yields different digests and a false
// result, so nondeterminism is caught rather than silently accepted.
func SHAMatches(parseA string, parseB string) bool {
	return parseA != "" && parseA == parseB
}

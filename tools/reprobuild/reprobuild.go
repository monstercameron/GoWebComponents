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

// BuildWasmSHA builds parsePackage for js/wasm with the release flags
// (-trimpath -ldflags "-s -w") into a throwaway file under parseTempDir and
// returns the lowercase hex SHA-256 of the resulting artifact.
func BuildWasmSHA(parseRepoRoot string, parsePackage string, parseTempDir string) (string, error) {
	parseOut := filepath.Join(parseTempDir, "repro.wasm")
	parseCmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", parseOut, parsePackage)
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

// SHAMatches reports whether two artifact digests are identical. It is the
// comparison the reproducibility check uses; a single differing byte (for
// example an embedded build timestamp) yields different digests and a false
// result, so nondeterminism is caught rather than silently accepted.
func SHAMatches(parseA string, parseB string) bool {
	return parseA != "" && parseA == parseB
}

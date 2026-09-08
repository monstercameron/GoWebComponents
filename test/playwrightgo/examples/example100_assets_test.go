//go:build playwrightgo

package playwrightgoexamples_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildExample100IsolatedAssets builds real application and worker bundles owned by one test.
func buildExample100IsolatedAssets(parseT *testing.T, parseRoot string, isAgent bool) string {
	parseT.Helper()
	parseDir := parseT.TempDir()
	parseTags := ""
	if isAgent {
		parseTags = "gwcagent"
	}
	for _, parseTarget := range []struct{ parsePackage, parseOutput string }{
		{"./examples/server/ai-chat-wizard/client", "app/chat.wasm"},
		{"./examples/server/ai-chat-wizard/client/backgroundworker", "worker/background-worker.wasm"},
	} {
		parseOutput := filepath.Join(parseDir, filepath.FromSlash(parseTarget.parseOutput))
		if parseErr := os.MkdirAll(filepath.Dir(parseOutput), 0o755); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		parseCommand := exec.Command("go", "build", "-tags", parseTags, "-o", parseOutput, parseTarget.parsePackage)
		parseCommand.Dir = parseRoot
		for _, parseEntry := range os.Environ() {
			parseKey, _, _ := strings.Cut(parseEntry, "=")
			switch strings.ToUpper(parseKey) {
			case "GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS":
				continue
			}
			parseCommand.Env = append(parseCommand.Env, parseEntry)
		}
		parseCommand.Env = append(parseCommand.Env, "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0", "GOFLAGS=")
		if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
			parseT.Fatalf("build isolated chat assets: %v\n%s", parseErr, parseOutput)
		}
	}
	return parseDir
}

package wasmdeps

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const forbiddenDepPrefix = "go.opentelemetry.io"

// getRepoRoot walks up from the test directory to the module root.
func getRepoRoot(parseT *testing.T) string {
	parseT.Helper()
	parseDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("getwd: %v", parseErr)
	}
	for {
		if _, parseStatErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseStatErr == nil {
			return parseDir
		}
		parseParent := filepath.Dir(parseDir)
		if parseParent == parseDir {
			parseT.Fatal("go.mod not found above test directory")
		}
		parseDir = parseParent
	}
}

// listDeps runs go list -deps for the given packages and returns the dependency import paths.
func listDeps(parseT *testing.T, parseGOOS string, parseGOARCH string, parsePackages []string) []string {
	parseT.Helper()
	if _, parseErr := exec.LookPath("go"); parseErr != nil {
		parseT.Skip("go tool is not available; skipping dependency guard")
	}
	parseArgs := append([]string{"list", "-deps", "-f", "{{.ImportPath}}"}, parsePackages...)
	parseCmd := exec.Command("go", parseArgs...)
	parseCmd.Dir = getRepoRoot(parseT)
	parseCmd.Env = os.Environ()
	if parseGOOS != "" {
		parseCmd.Env = append(parseCmd.Env, "GOOS="+parseGOOS, "GOARCH="+parseGOARCH)
	}
	parseOut, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("go list -deps failed (%s/%s): %v\n%s", parseGOOS, parseGOARCH, parseErr, parseOut)
	}
	return strings.Fields(string(parseOut))
}

func assertNoOTelDeps(parseT *testing.T, parseDeps []string) {
	parseT.Helper()
	for _, parseDep := range parseDeps {
		if strings.Contains(parseDep, forbiddenDepPrefix) {
			parseT.Errorf("forbidden dependency %q reachable from core packages", parseDep)
		}
	}
}

// TestWasmCoreHasNoOpenTelemetry keeps the browser build free of OpenTelemetry, whose root and
// propagation packages pull net/http and crypto/tls into the wasm binary.
func TestWasmCoreHasNoOpenTelemetry(parseT *testing.T) {
	parseDeps := listDeps(parseT, "js", "wasm", []string{
		"./ui", "./fetch", "./telemetry", "./internal/runtime", "./html", "./router",
		"./state", "./logging", "./interop", "./devtools",
	})
	assertNoOTelDeps(parseT, parseDeps)
}

// TestNativeCoreHasNoOpenTelemetry keeps the core library OpenTelemetry-free on the host too;
// adapters belong in separate opt-in packages.
func TestNativeCoreHasNoOpenTelemetry(parseT *testing.T) {
	parseDeps := listDeps(parseT, "", "", []string{
		"./ui", "./fetch", "./serverfn", "./logging", "./internal/runtime", "./telemetry",
	})
	assertNoOTelDeps(parseT, parseDeps)
}

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAgentBridgeReleaseArtifactHasNoAgentStrings(parseT *testing.T) {
	// Runs a full `go build -tags production` js/wasm build of the counter example
	// — a slow, toolchain-dependent step that can fail transiently and make the
	// default suite non-deterministic. Skip in -short so the fast dev-loop suite
	// stays deterministic; full CI runs (no -short) still enforce this security
	// check that the release artifact contains no agent-bridge strings.
	if testing.Short() {
		parseT.Skip("runs a full production wasm build; skipped in -short (still enforced in full CI)")
	}
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseOut := filepath.Join(parseT.TempDir(), "counter.wasm")
	parseCmd := exec.Command("go", "build", "-tags", "production", "-trimpath", "-ldflags=-s -w", "-o", parseOut, "./examples/public/counter")
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseCombined, parseErr2 := parseCmd.CombinedOutput()
	if parseErr2 != nil {
		parseT.Fatalf("build release wasm: %v\n%s", parseErr2, parseCombined)
	}
	parseRaw, parseErr3 := os.ReadFile(parseOut)
	if parseErr3 != nil {
		parseT.Fatalf("read release wasm: %v", parseErr3)
	}
	for _, parseForbidden := range []string{
		"__GWC_AGENT_BRIDGE",
		"gwc-agent-token",
		"/gwc-agent",
		"bridge.set-atom",
		"bridge.set-state",
		"gwcagent",
	} {
		if bytes.Contains(parseRaw, []byte(parseForbidden)) {
			parseT.Fatalf("release wasm contains agent bridge marker %q", parseForbidden)
		}
	}
}

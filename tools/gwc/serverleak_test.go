package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServerLeakFlagsClientOnlyImports proves the analyzer flags a server-only import in a
// browser-only (js && wasm) file, but NOT in a server-tagged file or a shared no-constraint
// file (no false positives).
func TestServerLeakFlagsClientOnlyImports(parseT *testing.T) {
	parseDir := parseT.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(parseDir, name), []byte(content), 0644); err != nil {
			parseT.Fatalf("write %s: %v", name, err)
		}
	}
	// Browser-only file importing os/exec -> LEAK.
	write("client.go", "//go:build js && wasm\n\npackage x\n\nimport _ \"os/exec\"\n")
	// Server-tagged file importing os/exec -> OK.
	write("server.go", "//go:build !js || !wasm\n\npackage x\n\nimport _ \"os/exec\"\n")
	// Shared (no constraint) file importing os/exec -> OK (not flagged; could be server).
	write("shared.go", "package x\n\nimport _ \"os/exec\"\n")
	// Browser-only file importing a fine package -> OK.
	write("client_ok.go", "//go:build js && wasm\n\npackage x\n\nimport _ \"strings\"\n")

	parseDiags := collectServerLeakDiagnostics(parseDir)
	if len(parseDiags) != 1 {
		parseT.Fatalf("expected exactly one leak (client.go), got %d: %#v", len(parseDiags), parseDiags)
	}
	if !strings.Contains(parseDiags[0].File, "client.go") || !strings.Contains(parseDiags[0].Message, "os/exec") {
		parseT.Fatalf("leak should name client.go + os/exec, got %#v", parseDiags[0])
	}
}

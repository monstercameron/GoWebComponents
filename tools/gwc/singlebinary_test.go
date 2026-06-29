package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateSingleBinaryServerIsValid proves the generated server scaffold is gofmt-clean
// Go that embeds the assets and serves them via wholestack.
func TestGenerateSingleBinaryServerIsValid(parseT *testing.T) {
	parseSrc, parseErr := generateSingleBinaryServer("dist", ":9000")
	if parseErr != nil {
		parseT.Fatalf("generate: %v", parseErr)
	}
	for _, parseWant := range []string{
		"package main",
		"//go:embed all:dist",
		"github.com/monstercameron/GoWebComponents/v4/wholestack",
		"wholestack.Handler(wholestack.Options{",
		`fs.Sub(assets, "dist")`,
		`addr := ":9000"`,
		"http.ListenAndServe(addr, handler)",
	} {
		if !strings.Contains(parseSrc, parseWant) {
			parseT.Fatalf("generated server missing %q:\n%s", parseWant, parseSrc)
		}
	}
}

// TestSingleBinaryServerCompiles is the FC5 e2e: the generated scaffold (plus a real assets
// dir) actually compiles into a binary. Built inside the repo module so wholestack resolves;
// removed afterward. Skipped in -short.
func TestSingleBinaryServerCompiles(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping single-binary compile in -short")
	}
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", ".."))
	if parseErr != nil {
		parseT.Fatalf("repo root: %v", parseErr)
	}
	parsePkgRel := "internal/gwcbundlesmoke"
	parseDir := filepath.Join(parseRepoRoot, parsePkgRel)
	if parseErr := os.MkdirAll(filepath.Join(parseDir, "dist"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir: %v", parseErr)
	}
	parseT.Cleanup(func() { _ = os.RemoveAll(parseDir) })

	if parseErr := os.WriteFile(filepath.Join(parseDir, "dist", "index.html"), []byte("<!doctype html><title>app</title>"), 0644); parseErr != nil {
		parseT.Fatalf("write index: %v", parseErr)
	}
	parseSrc, parseErr := generateSingleBinaryServer("dist", ":8080")
	if parseErr != nil {
		parseT.Fatalf("generate: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseDir, "main.go"), []byte(parseSrc), 0644); parseErr != nil {
		parseT.Fatalf("write main: %v", parseErr)
	}

	parseCmd := exec.Command("go", "build", "-o", filepath.Join(parseDir, "app.exe"), "./"+parsePkgRel)
	parseCmd.Dir = parseRepoRoot
	if parseOut, parseBuildErr := parseCmd.CombinedOutput(); parseBuildErr != nil {
		parseT.Fatalf("single-binary scaffold failed to compile:\n%s", parseOut)
	}
}

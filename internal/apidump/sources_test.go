package apidump

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestExtractSourcesMatchesFilesystemUnion verifies embedded and directory extraction retain the same cross-platform API surface.
func TestExtractSourcesMatchesFilesystemUnion(parseTest *testing.T) {
	parseSources := map[string][]byte{
		"types.go":        []byte("package fixture\ntype Config struct { Name string }\nvar Default *Config\nfunc private() {}\n"),
		"native.go":       []byte("//go:build windows\n\npackage fixture\nfunc NativeOnly() {}\nfunc Shared() {}\n"),
		"wasm.go":         []byte("//go:build js && wasm\n\npackage fixture\nfunc WasmOnly() {}\nfunc Shared() {}\n"),
		"ignored_test.go": []byte("this intentionally is not valid Go"),
		"ignored.txt":     []byte("nor is this"),
	}
	parseRoot := parseTest.TempDir()
	for parseName, parseData := range parseSources {
		if parseErr := os.WriteFile(filepath.Join(parseRoot, parseName), parseData, 0o644); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	parseDisk, parseErr := Extract(parseRoot)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseEmbedded, parseErr := ExtractSources(parseSources)
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if !slices.Equal(parseDisk, parseEmbedded) {
		parseTest.Fatalf("embedded extraction weakened API coverage: disk=%v embedded=%v", parseDisk, parseEmbedded)
	}
	for _, parseExpected := range []string{"func NativeOnly()", "func WasmOnly()", "func Shared()", "var Default *Config"} {
		if !slices.Contains(parseEmbedded, parseExpected) {
			parseTest.Fatalf("missing API %q: %v", parseExpected, parseEmbedded)
		}
	}
	CheckSources(parseTest, parseSources, []byte(strings.Join(parseDisk, "\r\n")+"\r\n"))
}

// TestExtractSourcesDetectsChangesAndMalformedSource prevents an embedded snapshot from becoming an inert golden comparison.
func TestExtractSourcesDetectsChangesAndMalformedSource(parseTest *testing.T) {
	parseBefore, parseErr := ExtractSources(map[string][]byte{"api.go": []byte("package fixture\nfunc Public(value int) string { return \"\" }\n")})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseAfter, parseErr := ExtractSources(map[string][]byte{"api.go": []byte("package fixture\nfunc Public(value string) string { return value }\n")})
	if parseErr != nil || slices.Equal(parseBefore, parseAfter) {
		parseTest.Fatalf("changed signature was not detected: %v, %v, %v", parseBefore, parseAfter, parseErr)
	}
	if _, parseErr = ExtractSources(map[string][]byte{"broken.go": []byte("package fixture\nfunc Broken(")}); parseErr == nil || !strings.Contains(parseErr.Error(), "broken.go") {
		parseTest.Fatalf("malformed production source did not fail with its filename: %v", parseErr)
	}
}

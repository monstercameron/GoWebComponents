package apidump

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func extractSource(parseT *testing.T, parseSrc string) []string {
	parseT.Helper()
	parseDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseDir, "sample.go"), []byte(parseSrc), 0o644); parseErr != nil {
		parseT.Fatalf("write source: %v", parseErr)
	}
	parseLines, parseErr := Extract(parseDir)
	if parseErr != nil {
		parseT.Fatalf("extract: %v", parseErr)
	}
	return parseLines
}

// TestExtractCapturesExplicitVarConstTypes pins that an explicitly-typed exported
// var/const carries its type into the baseline line (so a breaking type change is
// caught), while an untyped declaration stays name-only (conservative: the AST has
// no type to render without go/types). Funcs, methods, and struct types are also
// captured for regression coverage of the previously untested extractor.
func TestExtractCapturesExplicitVarConstTypes(parseT *testing.T) {
	parseLines := extractSource(parseT, `package sample

type Status int
type Config struct{ A int }

const StatusIdle Status = 0
const Untyped = 5

var Default *Config
var ErrX = "sentinel"

func Do(parseX int) string { return "" }
func (parseC *Config) Method() {}
`)

	parseWant := []string{
		"const StatusIdle Status", // typed const -> type captured
		"const Untyped",           // untyped const -> name only (no false line with a type)
		"var Default *Config",     // typed var -> type captured
		"var ErrX",                // untyped var -> name only
		"func (*Config) Method()", // method on exported receiver
	}
	for _, parseExpect := range parseWant {
		if !slices.Contains(parseLines, parseExpect) {
			parseT.Fatalf("expected extracted line %q, got %#v", parseExpect, parseLines)
		}
	}
	// The untyped const must NOT gain a spurious type suffix.
	if slices.Contains(parseLines, "const Untyped 5") || slices.Contains(parseLines, "const Untyped int") {
		parseT.Fatalf("untyped const must stay name-only, got %#v", parseLines)
	}
}

// TestExtractDetectsBreakingVarTypeChange pins the actual false-pass fix: two
// packages that differ ONLY in an exported var's type must produce different
// baselines. Before the fix both rendered "var Default", so the type change slid
// past the API-baseline gate silently.
func TestExtractDetectsBreakingVarTypeChange(parseT *testing.T) {
	parseBefore := extractSource(parseT, "package sample\ntype A struct{}\ntype B struct{}\nvar Default *A\n")
	parseAfter := extractSource(parseT, "package sample\ntype A struct{}\ntype B struct{}\nvar Default *B\n")

	if slices.Equal(parseBefore, parseAfter) {
		parseT.Fatalf("expected a var type change to alter the baseline, both were %#v", parseBefore)
	}
	if !slices.Contains(parseBefore, "var Default *A") || !slices.Contains(parseAfter, "var Default *B") {
		parseT.Fatalf("expected typed var lines; before=%#v after=%#v", parseBefore, parseAfter)
	}
}

package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(parseT *testing.T) string {
	parseT.Helper()
	parseDir, _ := os.Getwd()
	for range 8 {
		if _, parseErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseErr == nil {
			return parseDir
		}
		parseDir = filepath.Dir(parseDir)
	}
	parseT.Fatal("module root not found")
	return ""
}

// TestBuildDocumentParsesModuleList pins the decode logic against a fixture
// `go list -m -json all` stream: the main module is excluded, versioned deps
// become CycloneDX components with a golang purl, and they are sorted.
func TestBuildDocumentParsesModuleList(parseT *testing.T) {
	parseFixture := `{"Path":"example.com/me","Main":true}
{"Path":"golang.org/x/net","Version":"v0.55.0"}
{"Path":"github.com/yuin/goldmark","Version":"v1.7.13"}
{"Path":"example.com/noversion"}`
	parseDoc, parseErr := BuildDocument([]byte(parseFixture))
	if parseErr != nil {
		parseT.Fatalf("build: %v", parseErr)
	}
	if parseDoc.BOMFormat != "CycloneDX" || parseDoc.SpecVersion != "1.5" {
		parseT.Fatalf("unexpected BOM header: %+v", parseDoc)
	}
	if len(parseDoc.Components) != 2 {
		parseT.Fatalf("expected 2 components (main + no-version excluded), got %d", len(parseDoc.Components))
	}
	// Sorted by name: github.com/... before golang.org/...
	if parseDoc.Components[0].Name != "github.com/yuin/goldmark" {
		parseT.Fatalf("components not sorted by name: %+v", parseDoc.Components)
	}
	parseNet := parseDoc.Components[1]
	if parseNet.Name != "golang.org/x/net" || parseNet.Version != "v0.55.0" {
		parseT.Fatalf("unexpected x/net component: %+v", parseNet)
	}
	if parseNet.PURL != "pkg:golang/golang.org/x/net@v0.55.0" || parseNet.Type != "library" {
		parseT.Fatalf("bad purl/type: %+v", parseNet)
	}
}

// TestGenerateMatchesModuleGraph runs the real generator and checks the SBOM
// reflects the actual go.mod graph: it is valid CycloneDX, has a substantial
// component count, and contains a known dependency.
func TestGenerateMatchesModuleGraph(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("invokes go list; skipped in -short mode")
	}
	parseDoc, parseErr := Generate(repoRoot(parseT))
	if parseErr != nil {
		parseT.Fatalf("generate: %v", parseErr)
	}
	if parseDoc.BOMFormat != "CycloneDX" {
		parseT.Fatalf("not CycloneDX: %+v", parseDoc.BOMFormat)
	}
	if len(parseDoc.Components) < 50 {
		parseT.Fatalf("expected a substantial dependency graph, got %d components", len(parseDoc.Components))
	}
	parseFound := false
	for _, parseComponent := range parseDoc.Components {
		if parseComponent.Name == "golang.org/x/net" {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Fatal("expected golang.org/x/net in the SBOM")
	}
	// The document must round-trip as valid JSON.
	if _, parseErr := json.Marshal(parseDoc); parseErr != nil {
		parseT.Fatalf("marshal: %v", parseErr)
	}
}

func TestBuildDocumentRejectsMalformedModuleStream(parseT *testing.T) {
	_, parseErr := BuildDocument([]byte(`{"Path":"ok","Version":"v1.0.0"}` + "\n" + `{bad json}`))
	if parseErr == nil {
		parseT.Fatal("expected malformed module stream to fail")
	}
	if !strings.Contains(parseErr.Error(), "decode module list") {
		parseT.Fatalf("error should identify decode failure, got %v", parseErr)
	}
}

func TestWriteFileWritesIndentedCycloneDXJSON(parseT *testing.T) {
	parseRoot := repoRoot(parseT)
	parsePath := filepath.Join(parseT.TempDir(), "bom.json")
	if parseErr := WriteFile(parseRoot, parsePath); parseErr != nil {
		parseT.Fatalf("WriteFile: %v", parseErr)
	}
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("ReadFile: %v", parseErr)
	}
	if !strings.HasSuffix(string(parseData), "\n") || !strings.Contains(string(parseData), "\n  \"components\": [") {
		parseT.Fatalf("expected indented JSON with trailing newline, got prefix: %.80q", string(parseData))
	}
	var parseDoc Document
	if parseErr := json.Unmarshal(parseData, &parseDoc); parseErr != nil {
		parseT.Fatalf("written SBOM is not valid JSON: %v", parseErr)
	}
	if parseDoc.BOMFormat != "CycloneDX" || len(parseDoc.Components) == 0 {
		parseT.Fatalf("unexpected written SBOM: %+v", parseDoc)
	}
}

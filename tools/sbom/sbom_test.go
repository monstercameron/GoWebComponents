package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(parseT *testing.T) string {
	parseT.Helper()
	parseDir, _ := os.Getwd()
	for parseI := 0; parseI < 8; parseI++ {
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

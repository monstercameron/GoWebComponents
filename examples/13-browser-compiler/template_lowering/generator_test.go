package templatelowering

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedLandingMatchesTemplateOutput(parseT *testing.T) {
	parseTemplatePath := filepath.Join(".", "landing.template.html")
	parseGeneratedPath := filepath.Join(".", "generated_landing.go")

	parseGot, parseErr := GenerateFromFile(parseTemplatePath, TemplateConfig{
		PackageName: "templatelowering",
		StructName:  "LandingProps",
		FuncName:    "RenderLanding",
	})
	if parseErr != nil {
		parseT.Fatalf("GenerateFromFile() error = %v", parseErr)
	}

	parseWantBytes, parseErr := os.ReadFile(parseGeneratedPath)
	if parseErr != nil {
		parseT.Fatalf("ReadFile(%q) error = %v", parseGeneratedPath, parseErr)
	}
	if parseGot != string(parseWantBytes) {
		parseT.Fatalf("generated output drifted from checked-in Go file")
	}
}

func TestRenderLandingReturnsNode(parseT *testing.T) {
	parseNode := RenderLanding(LandingProps{
		Eyebrow:      "Compiler experiment",
		Headline:     "Optional template lowering",
		Summary:      "This lowers to ordinary Go builders.",
		PrimaryTag:   "Inspectable",
		SecondaryTag: "Optional",
	})
	if parseNode == nil {
		parseT.Fatal("RenderLanding() returned nil")
	}
}

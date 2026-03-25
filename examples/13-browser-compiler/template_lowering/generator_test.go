package templatelowering

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedLandingMatchesTemplateOutput(t *testing.T) {
	templatePath := filepath.Join(".", "landing.template.html")
	generatedPath := filepath.Join(".", "generated_landing.go")

	got, err := GenerateFromFile(templatePath, TemplateConfig{
		PackageName: "templatelowering",
		StructName:  "LandingProps",
		FuncName:    "RenderLanding",
	})
	if err != nil {
		t.Fatalf("GenerateFromFile() error = %v", err)
	}

	wantBytes, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", generatedPath, err)
	}
	if got != string(wantBytes) {
		t.Fatalf("generated output drifted from checked-in Go file")
	}
}

func TestRenderLandingReturnsNode(t *testing.T) {
	node := RenderLanding(LandingProps{
		Eyebrow:      "Compiler experiment",
		Headline:     "Optional template lowering",
		Summary:      "This lowers to ordinary Go builders.",
		PrimaryTag:   "Inspectable",
		SecondaryTag: "Optional",
	})
	if node == nil {
		t.Fatal("RenderLanding() returned nil")
	}
}

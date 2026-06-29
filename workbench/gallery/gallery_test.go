package gallery_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/workbench"
	"github.com/monstercameron/GoWebComponents/workbench/gallery"
)

// TestGalleryRendersStoryNamesAndPreviews renders the gallery headlessly and asserts each
// story's name and live preview appear — the same Story list RunStories smoke-tests.
func TestGalleryRendersStoryNamesAndPreviews(parseT *testing.T) {
	parseStories := []workbench.Story{
		{Name: "Primary Button", Render: func() ui.Node { return html.Button(html.Props{}, html.Text("Save")) }},
		{Name: "Danger Button", Render: func() ui.Node { return html.Button(html.Props{}, html.Text("Delete")) }},
	}

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, gallery.Gallery(parseStories...)); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}

	parseText := collectText(parseAdapter, parseRoot)
	for _, parseWant := range []string{"Primary Button", "Save", "Danger Button", "Delete"} {
		if !strings.Contains(parseText, parseWant) {
			parseT.Fatalf("gallery DOM missing %q; got %q", parseWant, parseText)
		}
	}
}

// TestGalleryToleratesStoryWithoutPreview proves a name-only story (nil/absent Render) still
// renders its label without panicking.
func TestGalleryToleratesStoryWithoutPreview(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, gallery.Gallery(workbench.Story{Name: "Doc Only"})); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	if !strings.Contains(collectText(parseAdapter, parseRoot), "Doc Only") {
		parseT.Fatal("expected the name-only story label to render")
	}
}

func collectText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectText(parseAdapter, parseChild)
	}
	return parseText
}

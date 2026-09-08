// Package workbench turns component examples ("stories") into headless smoke tests — the
// stories-as-tests half of the gwc workbench (FB5). A Story is a named example that renders
// a component; RunStories mounts each through the real reconciler into a mock DOM and
// reports any panic or render failure, so a component gallery doubles as a browserless test
// suite that runs under plain `go test`. The visual workbench gallery is a separate GUI
// consumer of the same Story list.
package workbench

import (
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Story is one named component example.
type Story struct {
	// Name identifies the story in failures and the gallery.
	Name string
	// Render produces the component tree to mount.
	Render func() ui.Node
}

// TestingT is the slice of *testing.T that RunStories needs, so it can be driven by a test
// or any compatible reporter. *testing.T satisfies it.
type TestingT interface {
	Helper()
	Errorf(parseFormat string, parseArgs ...any)
}

// RunStories renders every story through the native runtime into a mock DOM, reporting (via
// Errorf, so the run continues) any story that panics, renders a nil node, or fails to
// mount. It is the documented headless default: no browser, no harness, just `go test`.
func RunStories(parseT TestingT, parseStories ...Story) {
	parseT.Helper()
	for _, parseStory := range parseStories {
		runStory(parseT, parseStory)
	}
}

// runStory mounts one story, converting a panic into a reported failure.
func runStory(parseT TestingT, parseStory Story) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseT.Errorf("story %q panicked: %v", parseStory.Name, parseRecovered)
		}
	}()

	if parseStory.Render == nil {
		parseT.Errorf("story %q has no Render function", parseStory.Name)
		return
	}

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")

	parseNode := parseStory.Render()
	if parseNode == nil {
		parseT.Errorf("story %q rendered a nil node", parseStory.Name)
		return
	}
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Errorf("story %q failed to render: %v", parseStory.Name, parseErr)
	}
}

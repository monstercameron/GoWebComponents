package agentbridge

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestRenderDetachedDoesNotClobberAppRoot pins the root-isolation fix: rendering
// agent content into a second container via runtime.RenderDetached must NOT tear
// down the app's main root (which a plain RenderTo into a second container does,
// because the runtime has a single currentRoot). The app stays intact and the
// overlay renders independently.
func TestRenderDetachedDoesNotClobberAppRoot(t *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseAppC := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseAppC, "id", "app")
	parseOverlayC := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseOverlayC, "id", "overlay")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	// Render the app into #app through the global runtime.
	runtime.GetGlobalRuntime().RenderTo("#app",
		runtime.CreateElement("div", map[string]any{"id": "app-content"},
			runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": "APP"})))
	if len(parseAdapter.GetChildren(parseAppC)) == 0 {
		t.Fatal("precondition failed: app did not render into #app")
	}

	// Inject an overlay through the DETACHED runtime.
	if parseErr := runtime.RenderDetached("#overlay",
		runtime.CreateElement("div", map[string]any{"id": "overlay-content"},
			runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": "OVERLAY"}))); parseErr != nil {
		t.Fatalf("RenderDetached: %v", parseErr)
	}

	// The app must be untouched, and the overlay must have rendered.
	if len(parseAdapter.GetChildren(parseAppC)) == 0 {
		t.Fatal("RenderDetached clobbered the app root (#app emptied)")
	}
	if len(parseAdapter.GetChildren(parseOverlayC)) == 0 {
		t.Fatal("overlay did not render into #overlay")
	}

	// A second detached render into the same overlay also leaves the app intact.
	if parseErr := runtime.RenderDetached("#overlay",
		runtime.CreateElement("span", nil,
			runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": "OVERLAY-2"}))); parseErr != nil {
		t.Fatalf("RenderDetached second call: %v", parseErr)
	}
	if len(parseAdapter.GetChildren(parseAppC)) == 0 {
		t.Fatal("second RenderDetached clobbered the app root")
	}
}

// nodeTreeHasTag reports whether a host node with the given tag exists in the
// snapshot subtree.
func nodeTreeHasTag(parseNode *runtime.AgentNodeSnapshot, parseTag string) bool {
	if parseNode == nil {
		return false
	}
	if parseNode.Kind == "host" && parseNode.Name == parseTag {
		return true
	}
	for parseI := range parseNode.Children {
		if nodeTreeHasTag(&parseNode.Children[parseI], parseTag) {
			return true
		}
	}
	return false
}

// nodeTreeHasText reports whether any node's full Text contains the substring.
func nodeTreeHasText(parseNode *runtime.AgentNodeSnapshot, parseSub string) bool {
	if parseNode == nil {
		return false
	}
	if strings.Contains(parseNode.Text, parseSub) {
		return true
	}
	for parseI := range parseNode.Children {
		if nodeTreeHasText(&parseNode.Children[parseI], parseSub) {
			return true
		}
	}
	return false
}

// TestSnapshotIncludesDetachedRuntime pins the read fix: agent-injected content
// rendered via RenderDetached (which lives in a separate runtime, invisible to
// the global Inspect) now shows up in BuildAgentSnapshot's Detached list, so an
// agent can read it through the bridge.
func TestSnapshotIncludesDetachedRuntime(t *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseAppC := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseAppC, "id", "app2")
	parseOverlayC := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseOverlayC, "id", "overlay2")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	runtime.GetGlobalRuntime().RenderTo("#app2",
		runtime.CreateElement("div", map[string]any{"id": "app2-content"}))
	if parseErr := runtime.RenderDetached("#overlay2",
		runtime.CreateElement("article", nil,
			runtime.CreateElement("h1", nil,
				runtime.CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": "DETACHED HEADLINE"})))); parseErr != nil {
		t.Fatalf("RenderDetached: %v", parseErr)
	}

	parseSnap := runtime.BuildAgentSnapshot(runtime.GetGlobalRuntime(), 0, 0)

	// The main snapshot root must NOT contain the detached article (proves the
	// fix is additive, not a leak into the main tree).
	if nodeTreeHasTag(parseSnap.Root, "article") {
		t.Fatal("detached article leaked into the main snapshot root")
	}
	// The detached list must contain #overlay2 with the injected article+h1 AND
	// its readable text content.
	parseFound := false
	for parseI := range parseSnap.Detached {
		if parseSnap.Detached[parseI].Selector == "#overlay2" &&
			nodeTreeHasTag(parseSnap.Detached[parseI].Root, "article") &&
			nodeTreeHasTag(parseSnap.Detached[parseI].Root, "h1") &&
			nodeTreeHasText(parseSnap.Detached[parseI].Root, "DETACHED HEADLINE") {
			parseFound = true
		}
	}
	if !parseFound {
		t.Fatalf("snapshot.Detached did not contain the injected content + text; got %+v", parseSnap.Detached)
	}
}

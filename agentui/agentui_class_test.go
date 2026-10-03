package agentui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/agentui"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestClassPropIsConfinedToAgentUINamespace pins that agent-supplied utility
// classes cannot reach the host page's styles.
func TestClassPropIsConfinedToAgentUINamespace(parseT *testing.T) {
	parseCases := []struct{ parseIn, parseWant string }{
		{"ok", "agentui-card agentui-ok"},
		{"agentui-wide", "agentui-card agentui-wide"},
		{"fixed inset-0 z-50", "agentui-card agentui-fixed agentui-inset-0 agentui-z-50"},
		{"md:flex w-[100vw] top-1/2", "agentui-card"},
		{"", "agentui-card"},
	}
	for _, parseCase := range parseCases {
		parseNode, parseErr := agentui.DefaultRegistry().Render(agentui.Node{Type: "card", Props: map[string]string{"class": parseCase.parseIn}})
		if parseErr != nil {
			parseT.Fatalf("render %q: %v", parseCase.parseIn, parseErr)
		}
		parseHTML, parseRenderErr := ui.RenderToString(parseNode)
		if parseRenderErr != nil {
			parseT.Fatal(parseRenderErr)
		}
		if !strings.Contains(parseHTML, `class="`+parseCase.parseWant+`"`) {
			parseT.Errorf("class %q rendered %s, want class=%q", parseCase.parseIn, parseHTML, parseCase.parseWant)
		}
	}
}

package ssr_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	ssr "github.com/monstercameron/GoWebComponents/test/ssr"
	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestPreferredSSRWrappersMatchCompatibilityAliasBehavior(parseT *testing.T) {
	parseRoot := html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR parity"))
	parseBootstrap := ui.SSRBootstrap{}
	if parseErr := ui.RegisterBootstrapPayload(&parseBootstrap, "viewer", map[string]string{"name": "Cam"}); parseErr != nil {
		parseT.Fatalf("expected bootstrap registration to succeed, got %v", parseErr)
	}

	parsePreferredSnapshot := ssr.Render(parseT, parseRoot)
	parsePreferredViewer := ssr.RequirePayload[map[string]string](parseT, parseBootstrap, "viewer")
	parseCompatSnapshot := base.Render(parseT, parseRoot)
	parseCompatViewer := base.RequirePayload[map[string]string](parseT, parseBootstrap, "viewer")

	if parsePreferredSnapshot.HTML != parseCompatSnapshot.HTML {
		parseT.Fatalf("expected preferred wrapper and compatibility alias HTML to match, got preferred=%q compat=%q", parsePreferredSnapshot.HTML, parseCompatSnapshot.HTML)
	}
	if parsePreferredViewer.Value["name"] != parseCompatViewer.Value["name"] {
		parseT.Fatalf("expected preferred wrapper and compatibility alias payloads to match, got preferred=%+v compat=%+v", parsePreferredViewer.Value, parseCompatViewer.Value)
	}
}

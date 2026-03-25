package ssr_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	ssr "github.com/monstercameron/GoWebComponents/test/ssr"
	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestPreferredSSRWrappersMatchCompatibilityAliasBehavior(t *testing.T) {
	root := html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR parity"))
	bootstrap := ui.SSRBootstrap{}
	if err := ui.RegisterBootstrapPayload(&bootstrap, "viewer", map[string]string{"name": "Cam"}); err != nil {
		t.Fatalf("expected bootstrap registration to succeed, got %v", err)
	}

	preferredSnapshot := ssr.Render(t, root)
	preferredViewer := ssr.RequirePayload[map[string]string](t, bootstrap, "viewer")
	compatSnapshot := base.Render(t, root)
	compatViewer := base.RequirePayload[map[string]string](t, bootstrap, "viewer")

	if preferredSnapshot.HTML != compatSnapshot.HTML {
		t.Fatalf("expected preferred wrapper and compatibility alias HTML to match, got preferred=%q compat=%q", preferredSnapshot.HTML, compatSnapshot.HTML)
	}
	if preferredViewer.Value["name"] != compatViewer.Value["name"] {
		t.Fatalf("expected preferred wrapper and compatibility alias payloads to match, got preferred=%+v compat=%+v", preferredViewer.Value, compatViewer.Value)
	}
}

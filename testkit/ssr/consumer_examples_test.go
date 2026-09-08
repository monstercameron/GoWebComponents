package ssr_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestConsumerSSRPattern_SnapshotAndPayload(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{}
	if parseErr := ui.RegisterBootstrapPayload(&parseBootstrap, "viewer", map[string]string{"name": "Cam"}); parseErr != nil {
		parseT.Fatalf("expected bootstrap registration to succeed, got %v", parseErr)
	}

	parseSnapshot := ssr.Render(parseT, html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR page")))
	parseViewer := ssr.RequirePayload[map[string]string](parseT, parseBootstrap, "viewer")

	if !parseSnapshot.Contains("SSR page") {
		parseT.Fatalf("expected snapshot to contain rendered HTML, got %q", parseSnapshot.HTML)
	}
	if parseViewer.Value["name"] != "Cam" {
		parseT.Fatalf("expected typed bootstrap payload, got %+v", parseViewer.Value)
	}
}

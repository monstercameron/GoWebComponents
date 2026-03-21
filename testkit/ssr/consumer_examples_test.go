package ssr_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestConsumerSSRPattern_SnapshotAndPayload(t *testing.T) {
	bootstrap := ui.SSRBootstrap{}
	if err := ui.RegisterBootstrapPayload(&bootstrap, "viewer", map[string]string{"name": "Cam"}); err != nil {
		t.Fatalf("expected bootstrap registration to succeed, got %v", err)
	}

	snapshot := ssr.Render(t, html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR page")))
	viewer := ssr.RequirePayload[map[string]string](t, bootstrap, "viewer")

	if !snapshot.Contains("SSR page") {
		t.Fatalf("expected snapshot to contain rendered HTML, got %q", snapshot.HTML)
	}
	if viewer.Value["name"] != "Cam" {
		t.Fatalf("expected typed bootstrap payload, got %+v", viewer.Value)
	}
}

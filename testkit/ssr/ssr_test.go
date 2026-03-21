package ssr

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRenderSnapshotsHTML(t *testing.T) {
	snapshot := Render(t, html.Div(html.Props{ID: "ssr-root"}, html.Text("SSR Ready")))
	if !snapshot.Contains("SSR Ready") || !snapshot.Contains("id=\"ssr-root\"") {
		t.Fatalf("expected snapshot HTML to contain rendered markup, got %q", snapshot.HTML)
	}
}

func TestRequirePayloadReadsTypedBootstrapValue(t *testing.T) {
	bootstrap := ui.SSRBootstrap{}
	if err := ui.RegisterBootstrapPayload(&bootstrap, "profile", map[string]string{"name": "Cam"}); err != nil {
		t.Fatalf("expected bootstrap registration to succeed, got %v", err)
	}
	profile := RequirePayload[map[string]string](t, bootstrap, "profile")
	if profile.Value["name"] != "Cam" {
		t.Fatalf("expected typed bootstrap payload, got %+v", profile.Value)
	}
}

//go:build !js || !wasm
// +build !js !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type greetingProps struct {
	Name string
}

func greeting(props greetingProps) ui.Node {
	return html.Section(html.Props{ID: "greeting"},
		html.H1(html.Props{}, html.Text("Hello "+props.Name)),
		html.Input(html.Props{ID: "email", Disabled: true}),
	)
}

func TestRenderToStringPublicSSRSurface(t *testing.T) {
	node := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	want := `<section id="greeting"><h1>Hello Server</h1><input disabled id="email"></section>`
	if markup != want {
		t.Fatalf("unexpected markup\nwant: %s\ngot:  %s", want, markup)
	}
}

func TestHydrateUnsupportedOnServer(t *testing.T) {
	_, err := ui.Hydrate(ui.Text("hello"), "#app")
	if err == nil {
		t.Fatal("expected Hydrate to be unavailable on non-js/wasm builds")
	}
}

func TestReadBootstrapReferenceUnsupportedOnServer(t *testing.T) {
	_, err := ui.ReadBootstrapReference(ui.SSRBootstrapReference{URL: "/bootstrap.cbor", Format: ui.SSRBootstrapFormatCBOR})
	if err == nil {
		t.Fatal("expected ReadBootstrapReference to be unavailable on non-js/wasm builds")
	}
}

//go:build !js || !wasm
// +build !js !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func benchmarkBootstrapPayload() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   "/products/42",
			Query:  map[string][]string{"tab": {"specs"}, "filter": {"active"}},
			Params: map[string]string{"id": "42"},
		},
		Atoms: map[string]interface{}{
			"theme":    "dark",
			"locale":   "en-US",
			"cartSize": 3,
		},
		Data: map[string]interface{}{
			"title":   "Widget",
			"price":   19.95,
			"inStock": true,
		},
		IDSeed: 17,
	}
}

func benchmarkRenderNode() ui.Node {
	return ui.CreateElement(func() ui.Node {
		return html.Main(html.Props{ID: "app"},
			html.Section(html.Props{Class: "hero"},
				html.H1(html.Props{}, html.Text("Server Rendered")),
				html.P(html.Props{}, html.Text("Bootstrap performance baseline")),
				html.Input(html.Props{ID: "email", Disabled: true}),
			),
		)
	})
}

func BenchmarkRenderToStringPublicSSRSurface(b *testing.B) {
	node := benchmarkRenderNode()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		markup, err := ui.RenderToString(node)
		if err != nil {
			b.Fatal(err)
		}
		if len(markup) == 0 {
			b.Fatal("expected markup")
		}
	}
}

func BenchmarkMarshalSSRBootstrapJSON(b *testing.B) {
	payload := benchmarkBootstrapPayload()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := ui.MarshalSSRBootstrap(payload)
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("expected encoded JSON payload")
		}
	}
}

func BenchmarkMarshalSSRBootstrapBinary(b *testing.B) {
	payload := benchmarkBootstrapPayload()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := ui.MarshalSSRBootstrapBinary(payload)
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("expected encoded binary payload")
		}
	}
}

func BenchmarkUnmarshalSSRBootstrapJSON(b *testing.B) {
	encoded, err := ui.MarshalSSRBootstrap(benchmarkBootstrapPayload())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, err := ui.UnmarshalSSRBootstrap(encoded)
		if err != nil {
			b.Fatal(err)
		}
		if payload.Route.Path == "" {
			b.Fatal("expected decoded JSON payload")
		}
	}
}

func BenchmarkUnmarshalSSRBootstrapBinary(b *testing.B) {
	encoded, err := ui.MarshalSSRBootstrapBinary(benchmarkBootstrapPayload())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, err := ui.UnmarshalSSRBootstrapBinary(encoded)
		if err != nil {
			b.Fatal(err)
		}
		if payload.Route.Path == "" {
			b.Fatal("expected decoded binary payload")
		}
	}
}

func BenchmarkRenderBootstrapReferenceScript(b *testing.B) {
	ref := ui.SSRBootstrapReference{URL: "/bootstrap.cbor", Format: ui.SSRBootstrapFormatCBOR}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		script, err := ui.RenderBootstrapReferenceScript(ref, "")
		if err != nil {
			b.Fatal(err)
		}
		if len(script) == 0 {
			b.Fatal("expected reference script")
		}
	}
}

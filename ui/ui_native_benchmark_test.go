//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func benchmarkBootstrapPayload() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   "/products/42",
			Query:  map[string][]string{"tab": {"specs"}, "filter": {"active"}},
			Params: map[string]string{"id": "42"},
		},
		Atoms: map[string]any{
			"theme":    "dark",
			"locale":   "en-US",
			"cartSize": 3,
		},
		Data: map[string]any{
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

func BenchmarkRenderToStringPublicSSRSurface(parseB *testing.B) {
	parseNode := benchmarkRenderNode()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseMarkup, parseErr := ui.RenderToString(parseNode)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseMarkup) == 0 {
			parseB.Fatal("expected markup")
		}
	}
}

func BenchmarkMarshalSSRBootstrapJSON(parseB *testing.B) {
	parsePayload := benchmarkBootstrapPayload()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseData, parseErr := ui.MarshalSSRBootstrap(parsePayload)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseData) == 0 {
			parseB.Fatal("expected encoded JSON payload")
		}
	}
}

func BenchmarkMarshalSSRBootstrapBinary(parseB *testing.B) {
	parsePayload := benchmarkBootstrapPayload()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseData, parseErr := ui.MarshalSSRBootstrapBinary(parsePayload)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseData) == 0 {
			parseB.Fatal("expected encoded binary payload")
		}
	}
}

func BenchmarkUnmarshalSSRBootstrapJSON(parseB *testing.B) {
	parseEncoded, parseErr := ui.MarshalSSRBootstrap(benchmarkBootstrapPayload())
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parsePayload, parseErr2 := ui.UnmarshalSSRBootstrap(parseEncoded)
		if parseErr2 != nil {
			parseB.Fatal(parseErr2)
		}
		if parsePayload.Route.Path == "" {
			parseB.Fatal("expected decoded JSON payload")
		}
	}
}

func BenchmarkUnmarshalSSRBootstrapBinary(parseB *testing.B) {
	parseEncoded, parseErr := ui.MarshalSSRBootstrapBinary(benchmarkBootstrapPayload())
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parsePayload, parseErr2 := ui.UnmarshalSSRBootstrapBinary(parseEncoded)
		if parseErr2 != nil {
			parseB.Fatal(parseErr2)
		}
		if parsePayload.Route.Path == "" {
			parseB.Fatal("expected decoded binary payload")
		}
	}
}

func BenchmarkRenderBootstrapReferenceScript(parseB *testing.B) {
	parseRef := ui.SSRBootstrapReference{URL: "/bootstrap.cbor", Format: ui.SSRBootstrapFormatCBOR}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseScript, parseErr := ui.RenderBootstrapReferenceScript(parseRef, "")
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseScript) == 0 {
			parseB.Fatal("expected reference script")
		}
	}
}

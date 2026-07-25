//go:build !js || !wasm

package main

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func BenchmarkSSRRoutingDemoRenderToString(parseB *testing.B) {
	parseNode := renderDemoShell(defaultServerView())
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseMarkup, parseErr := ui.RenderToString(parseNode)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseMarkup) == 0 {
			parseB.Fatal("expected SSR markup")
		}
	}
}

func BenchmarkSSRRoutingDemoMarshalBootstrapJSON(parseB *testing.B) {
	parsePayload := defaultBootstrapPayload()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseEncoded, parseErr := ui.MarshalSSRBootstrap(parsePayload)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseEncoded) == 0 {
			parseB.Fatal("expected bootstrap payload")
		}
	}
}

func BenchmarkSSRRoutingDemoRenderReferenceScript(parseB *testing.B) {
	parseBootstrapReference := ui.SSRBootstrapReference{URL: "bootstrap.json", Format: ui.SSRBootstrapFormatJSON}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseScript, parseErr := ui.RenderBootstrapReferenceScript(parseBootstrapReference, "")
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseScript) == 0 {
			parseB.Fatal("expected reference script")
		}
	}
}

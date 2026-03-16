package main

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkSSRRoutingDemoRenderToString(b *testing.B) {
	node := renderDemoShell(defaultServerView())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		markup, err := ui.RenderToString(node)
		if err != nil {
			b.Fatal(err)
		}
		if len(markup) == 0 {
			b.Fatal("expected SSR markup")
		}
	}
}

func BenchmarkSSRRoutingDemoMarshalBootstrapJSON(b *testing.B) {
	payload := defaultBootstrapPayload()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded, err := ui.MarshalSSRBootstrap(payload)
		if err != nil {
			b.Fatal(err)
		}
		if len(encoded) == 0 {
			b.Fatal("expected bootstrap payload")
		}
	}
}

func BenchmarkSSRRoutingDemoRenderReferenceScript(b *testing.B) {
	ref := ui.SSRBootstrapReference{URL: "bootstrap.json", Format: ui.SSRBootstrapFormatJSON}
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

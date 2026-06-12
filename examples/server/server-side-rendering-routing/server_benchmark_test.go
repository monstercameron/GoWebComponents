//go:build !js || !wasm

package main

import (
	"net/url"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkServerSSRResolveRouteDocs(parseB *testing.B) {
	parseQuery := url.Values{"tab": {serverTabLoader}, "refresh": {"2"}}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseResolved := resolveRoute("/docs/"+serverGuideSectionSSR, parseQuery)
		if parseResolved.Status != 200 || parseResolved.View.SectionID == "" {
			parseB.Fatal("expected docs route to resolve")
		}
	}
}

func BenchmarkServerSSRRenderDocumentBody(parseB *testing.B) {
	parseResolved := resolveRoute("/docs/"+serverGuideSectionSSR, url.Values{"tab": {serverTabLoader}})
	parseNode := renderDemoShell(parseResolved.View)
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

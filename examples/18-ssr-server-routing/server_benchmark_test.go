//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"net/url"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkServerSSRResolveRouteDocs(b *testing.B) {
	query := url.Values{"tab": {serverTabLoader}, "refresh": {"2"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved := resolveRoute("/docs/"+serverGuideSectionSSR, query)
		if resolved.Status != 200 || resolved.View.SectionID == "" {
			b.Fatal("expected docs route to resolve")
		}
	}
}

func BenchmarkServerSSRRenderDocumentBody(b *testing.B) {
	resolved := resolveRoute("/docs/"+serverGuideSectionSSR, url.Values{"tab": {serverTabLoader}})
	node := renderDemoShell(resolved.View)
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

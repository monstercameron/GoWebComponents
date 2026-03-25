//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func benchmarkNestedHashRouter() *Router {
	parseR := NewHashRouter()
	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/dashboard/settings", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/dashboard/settings/profile", func(parseProps3 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("profile"))
	})
	parseR.GoRegisterRoute("/docs", func(parseProps4 Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/docs/getting-started", func(parseProps5 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs"))
	})
	return parseR
}

func BenchmarkResolveRouteStackNestedLayout(parseB *testing.B) {
	installRouterBrowserEnv(parseB)
	parseR := benchmarkNestedHashRouter()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseResolved := parseR.resolveRouteStack("/dashboard/settings/profile")
		if !parseResolved.found || len(parseResolved.routes) != 3 {
			parseB.Fatal("expected 3-level resolved route stack")
		}
	}
}

func BenchmarkCurrentNestedLayoutRoute(parseB *testing.B) {
	installRouterBrowserEnv(parseB)
	parseR := benchmarkNestedHashRouter()
	globalRouter = parseR
	js.Global().Get("location").Set("hash", "/dashboard/settings/profile")
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseElem := parseR.Current()
		if parseElem == nil {
			parseB.Fatal("expected nested route element")
		}
	}
}

func BenchmarkEvaluateNavigationNestedLayouts(parseB *testing.B) {
	installRouterBrowserEnv(parseB)
	parseR := benchmarkNestedHashRouter()
	globalRouter = parseR
	js.Global().Get("location").Set("hash", "/dashboard/settings/profile")
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseTarget, parseOk := parseR.evaluateNavigation("/docs/getting-started")
		if !parseOk || parseTarget != "/docs/getting-started" {
			parseB.Fatal("expected nested navigation evaluation to succeed")
		}
	}
}

//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func benchmarkNestedHashRouter() *Router {
	r := NewHashRouter()
	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		return runtime.Div(nil, Outlet())
	}, Options{Layout: true})
	r.GoRegisterRoute("/dashboard/settings", func(props Attrs) *Element {
		return runtime.Div(nil, Outlet())
	}, Options{Layout: true})
	r.GoRegisterRoute("/dashboard/settings/profile", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("profile"))
	})
	r.GoRegisterRoute("/docs", func(props Attrs) *Element {
		return runtime.Div(nil, Outlet())
	}, Options{Layout: true})
	r.GoRegisterRoute("/docs/getting-started", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs"))
	})
	return r
}

func BenchmarkResolveRouteStackNestedLayout(b *testing.B) {
	installRouterBrowserEnv(b)
	r := benchmarkNestedHashRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		resolved := r.resolveRouteStack("/dashboard/settings/profile")
		if !resolved.found || len(resolved.routes) != 3 {
			b.Fatal("expected 3-level resolved route stack")
		}
	}
}

func BenchmarkCurrentNestedLayoutRoute(b *testing.B) {
	installRouterBrowserEnv(b)
	r := benchmarkNestedHashRouter()
	globalRouter = r
	js.Global().Get("location").Set("hash", "/dashboard/settings/profile")
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		elem := r.Current()
		if elem == nil {
			b.Fatal("expected nested route element")
		}
	}
}

func BenchmarkEvaluateNavigationNestedLayouts(b *testing.B) {
	installRouterBrowserEnv(b)
	r := benchmarkNestedHashRouter()
	globalRouter = r
	js.Global().Get("location").Set("hash", "/dashboard/settings/profile")
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		target, ok := r.evaluateNavigation("/docs/getting-started")
		if !ok || target != "/docs/getting-started" {
			b.Fatal("expected nested navigation evaluation to succeed")
		}
	}
}

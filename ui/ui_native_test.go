//go:build !js || !wasm
// +build !js !wasm

package ui_test

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type greetingProps struct {
	Name string
}

type reactiveRegionTestSource struct {
	id string
}

func (s reactiveRegionTestSource) ReactiveRegionSourceIDs() []string {
	if s.id == "" {
		return nil
	}
	return []string{s.id}
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

func TestRenderToStringObservedReportsMetrics(t *testing.T) {
	node := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	var observed ui.SSRObservation

	markup, err := ui.RenderToStringObserved(node, ui.SSRObservabilityOptions{
		CorrelationID: "req-render",
		OnEvent: func(event ui.SSRObservation) {
			observed = event
		},
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if markup == "" {
		t.Fatal("expected rendered markup")
	}
	if observed.Name != "ssr.render" || observed.Phase != "finish" {
		t.Fatalf("expected ssr render observation, got %+v", observed)
	}
	if observed.CorrelationID != "req-render" {
		t.Fatalf("expected correlation id to flow into render event, got %+v", observed)
	}
	if observed.Render == nil || observed.Render.DurationNs < 0 {
		t.Fatalf("expected non-negative render metrics, got %+v", observed)
	}
	if observed.Bootstrap != nil || observed.Hydration != nil {
		t.Fatalf("expected render-only observation, got %+v", observed)
	}
}

func TestObserveSSRReceivesBootstrapMetrics(t *testing.T) {
	var observed ui.SSRObservation
	unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
		if event.Name == "ssr.bootstrap" {
			observed = event
		}
	})
	defer unsubscribe()

	_, err := ui.RenderBootstrapScript(ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/home"}}, "")
	if err != nil {
		t.Fatalf("unexpected bootstrap render error: %v", err)
	}
	if observed.Bootstrap == nil {
		t.Fatalf("expected bootstrap observation, got %+v", observed)
	}
	if observed.Bootstrap.Format != ui.SSRBootstrapFormatJSON {
		t.Fatalf("expected json bootstrap format, got %+v", observed)
	}
	if observed.Bootstrap.PayloadBytes <= 0 || observed.Bootstrap.ScriptBytes <= 0 {
		t.Fatalf("expected positive bootstrap sizes, got %+v", observed)
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

func TestAsyncBoundaryAndLazyOnServer(t *testing.T) {
	fallback := ui.Text("loading")
	content := ui.Text("ready")
	if got := ui.AsyncBoundary(ui.AsyncBoundaryProps{Content: content}); got != content {
		t.Fatal("expected AsyncBoundary to return content when not pending on server")
	}
	if got := ui.AsyncBoundary(ui.AsyncBoundaryProps{Pending: true, Fallback: fallback}); got != fallback {
		t.Fatal("expected AsyncBoundary to return fallback when pending on server")
	}

	lazy := ui.UseLazyNode(func(ctx context.Context) (ui.Node, error) {
		return ui.Text("resolved"), nil
	})
	state := lazy.Get()
	if !state.Ready || state.Error != nil || state.Node == nil {
		t.Fatalf("expected server lazy node to resolve synchronously, got %+v", state)
	}

	got := ui.Lazy(ui.LazyProps{
		Loader: func(ctx context.Context) (ui.Node, error) {
			return ui.Text("resolved"), nil
		},
		Fallback: fallback,
	})
	markup, err := ui.RenderToString(got)
	if err != nil {
		t.Fatalf("unexpected async boundary render error: %v", err)
	}
	if markup != `resolved` {
		t.Fatalf("expected lazy server render to resolve content, got %q", markup)
	}
}

func TestContextProviderWrapperRendersChildrenOnServer(t *testing.T) {
	theme := ui.CreateContext("light")
	node := ui.CreateElement(theme.Provider, ui.ContextProviderProps[string]{
		Value: "dark",
		Child: html.P(html.Props{}, html.Text("context shell")),
	})

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected provider render error: %v", err)
	}
	if markup != `<p>context shell</p>` {
		t.Fatalf("expected provider wrapper to render child subtree, got %q", markup)
	}
}

func TestPortalRendersChildrenInlineOnServer(t *testing.T) {
	node := ui.Portal(ui.PortalProps{
		Target: ui.PortalTarget{Selector: "#overlay-root"},
		Child:  html.P(html.Props{}, html.Text("portal body")),
	})

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected portal render error: %v", err)
	}
	if markup != `<p>portal body</p>` {
		t.Fatalf("expected server portal fallback to render child inline, got %q", markup)
	}
}

func TestReactiveRegionRendersChildrenOnServer(t *testing.T) {
	node := ui.ReactiveRegion(func() ui.Node {
		return html.Span(html.Props{ID: "status"}, html.Text("server region"))
	}, reactiveRegionTestSource{id: "count"})

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected reactive region render error: %v", err)
	}
	if markup != `<span id="status">server region</span>` {
		t.Fatalf("expected reactive region SSR to render child subtree, got %q", markup)
	}
}

func TestErrorBoundaryRendersFallbackOnServer(t *testing.T) {
	boom := func() ui.Node {
		panic("server boundary")
	}
	node := ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ErrorFallback: func(err error, reset func()) ui.Node {
			if err == nil || err.Error() != "server boundary" {
				t.Fatalf("unexpected server boundary error: %v", err)
			}
			return html.P(html.Props{}, html.Text("caught on server"))
		},
		Child: ui.CreateElement(boom),
	})

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error boundary render error: %v", err)
	}
	if markup != `<p>caught on server</p>` {
		t.Fatalf("unexpected server boundary fallback markup: %q", markup)
	}
}

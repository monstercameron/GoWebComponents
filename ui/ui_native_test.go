//go:build !js || !wasm

package ui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type greetingProps struct {
	Name string
}

type reactiveRegionTestSource struct {
	id string
}

func (parseS reactiveRegionTestSource) ReactiveRegionSourceIDs() []string {
	if parseS.id == "" {
		return nil
	}
	return []string{parseS.id}
}

func greeting(parseProps greetingProps) ui.Node {
	return html.Section(html.Props{ID: "greeting"},
		html.H1(html.Props{}, html.Text("Hello "+parseProps.Name)),
		html.Input(html.Props{ID: "email", Disabled: true}),
	)
}

func TestRenderToStringPublicSSRSurface(parseT *testing.T) {
	parseNode := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseWant := `<section id="greeting"><h1>Hello Server</h1><input disabled id="email"></section>`
	if parseMarkup != parseWant {
		parseT.Fatalf("unexpected markup\nwant: %s\ngot:  %s", parseWant, parseMarkup)
	}
}

func TestRenderToStringObservedReportsMetrics(parseT *testing.T) {
	parseNode := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	var parseObserved ui.SSRObservation

	parseMarkup, parseErr := ui.RenderToStringObserved(parseNode, ui.SSRObservabilityOptions{
		CorrelationID: "req-render",
		OnEvent: func(parseEvent ui.SSRObservation) {
			parseObserved = parseEvent
		},
	})
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if parseMarkup == "" {
		parseT.Fatal("expected rendered markup")
	}
	if parseObserved.Name != "ssr.render" || parseObserved.Phase != "finish" {
		parseT.Fatalf("expected ssr render observation, got %+v", parseObserved)
	}
	if parseObserved.CorrelationID != "req-render" {
		parseT.Fatalf("expected correlation id to flow into render event, got %+v", parseObserved)
	}
	if parseObserved.Render == nil || parseObserved.Render.DurationNs < 0 {
		parseT.Fatalf("expected non-negative render metrics, got %+v", parseObserved)
	}
	if parseObserved.Bootstrap != nil || parseObserved.Hydration != nil {
		parseT.Fatalf("expected render-only observation, got %+v", parseObserved)
	}
}

func TestRenderToStreamPublicSSRSurface(parseT *testing.T) {
	parseNode := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	var parseBuffer bytes.Buffer
	var parseChunks []ui.SSRStreamChunk

	parseErr := ui.RenderToStream(context.Background(), &parseBuffer, parseNode, ui.SSRStreamOptions{
		OnChunk: func(parseChunk ui.SSRStreamChunk) {
			parseChunks = append(parseChunks, parseChunk)
		},
	})
	if parseErr != nil {
		parseT.Fatalf("unexpected stream render error: %v", parseErr)
	}

	parseWant := `<section id="greeting"><h1>Hello Server</h1><input disabled id="email"></section>`
	if parseBuffer.String() != parseWant {
		parseT.Fatalf("unexpected stream markup\nwant: %s\ngot:  %s", parseWant, parseBuffer.String())
	}
	if len(parseChunks) != 1 || parseChunks[0].Kind != ui.SSRStreamChunkShell || parseChunks[0].HTML != parseWant {
		parseT.Fatalf("expected one shell stream chunk, got %+v", parseChunks)
	}
}

func TestRenderToStreamObservedReportsMetrics(parseT *testing.T) {
	parseNode := ui.CreateElement(greeting, greetingProps{Name: "Server"})
	var parseBuffer bytes.Buffer
	var parseObserved ui.SSRObservation

	parseErr := ui.RenderToStreamObserved(context.Background(), &parseBuffer, parseNode, ui.SSRObservabilityOptions{
		CorrelationID: "req-stream",
		OnEvent: func(parseEvent ui.SSRObservation) {
			parseObserved = parseEvent
		},
	})
	if parseErr != nil {
		parseT.Fatalf("unexpected stream render error: %v", parseErr)
	}
	if parseBuffer.String() == "" {
		parseT.Fatal("expected streamed markup")
	}
	if parseObserved.Name != "ssr.render" || parseObserved.Phase != "finish" {
		parseT.Fatalf("expected ssr render observation, got %+v", parseObserved)
	}
	if parseObserved.CorrelationID != "req-stream" {
		parseT.Fatalf("expected stream correlation id, got %+v", parseObserved)
	}
	if parseObserved.Render == nil || parseObserved.Render.DurationNs < 0 {
		parseT.Fatalf("expected stream render metrics, got %+v", parseObserved)
	}
}

func TestRenderToStreamAsyncBoundaryEmitsBoundaryChunk(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseContent := ui.CreateElement(func() ui.Node {
		ui.SuspendUntil(parseDone, "stream user")
		return html.Strong(html.Props{}, html.Text("ready"))
	})
	parseRoot := ui.AsyncBoundary(ui.AsyncBoundaryProps{
		Fallback: html.Span(html.Props{}, html.Text("loading")),
		Content:  parseContent,
	})

	parseChunks := make(chan ui.SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- ui.RenderToStream(context.Background(), &parseBuffer, parseRoot, ui.SSRStreamOptions{
			OnChunk: func(parseChunk ui.SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveUIStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != ui.SSRStreamChunkShell || !strings.Contains(parseShell.HTML, "loading") || strings.Contains(parseShell.HTML, "ready") {
		parseT.Fatalf("expected fallback shell, got %+v", parseShell)
	}

	close(parseDone)
	parseBoundary := receiveUIStreamTestChunk(parseT, parseChunks)
	if parseBoundary.Kind != ui.SSRStreamChunkBoundary || !strings.Contains(parseBoundary.HTML, "<strong>ready</strong>") {
		parseT.Fatalf("expected ready boundary chunk, got %+v", parseBoundary)
	}
	if parseErr := receiveUIStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream returned error: %v", parseErr)
	}
}

func receiveUIStreamTestChunk(parseT *testing.T, parseChunks <-chan ui.SSRStreamChunk) ui.SSRStreamChunk {
	parseT.Helper()
	select {
	case parseChunk := <-parseChunks:
		return parseChunk
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for UI SSR stream chunk")
		return ui.SSRStreamChunk{}
	}
}

func receiveUIStreamTestError(parseT *testing.T, parseErrs <-chan error) error {
	parseT.Helper()
	select {
	case parseErr := <-parseErrs:
		return parseErr
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for UI SSR stream completion")
		return nil
	}
}

func TestObserveSSRReceivesBootstrapMetrics(parseT *testing.T) {
	var parseObserved ui.SSRObservation
	parseSub := ui.RegisterSSRObserver(func(parseEvent ui.SSRObservation) {
		if parseEvent.Name == "ssr.bootstrap" {
			parseObserved = parseEvent
		}
	})
	defer parseSub.Cancel()

	_, parseErr := ui.RenderBootstrapScript(ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/home"}}, "")
	if parseErr != nil {
		parseT.Fatalf("unexpected bootstrap render error: %v", parseErr)
	}
	if parseObserved.Bootstrap == nil {
		parseT.Fatalf("expected bootstrap observation, got %+v", parseObserved)
	}
	if parseObserved.Bootstrap.Format != ui.SSRBootstrapFormatJSON {
		parseT.Fatalf("expected json bootstrap format, got %+v", parseObserved)
	}
	if parseObserved.Bootstrap.PayloadBytes <= 0 || parseObserved.Bootstrap.ScriptBytes <= 0 {
		parseT.Fatalf("expected positive bootstrap sizes, got %+v", parseObserved)
	}
}

func TestHydrateUnsupportedOnServer(parseT *testing.T) {
	_, parseErr := ui.Hydrate(ui.Text("hello"), "#app")
	if parseErr == nil {
		parseT.Fatal("expected Hydrate to be unavailable on non-js/wasm builds")
	}
}

func TestReadBootstrapReferenceUnsupportedOnServer(parseT *testing.T) {
	_, parseErr := ui.ReadBootstrapReference(ui.SSRBootstrapReference{URL: "/bootstrap.cbor", Format: ui.SSRBootstrapFormatCBOR})
	if parseErr == nil {
		parseT.Fatal("expected ReadBootstrapReference to be unavailable on non-js/wasm builds")
	}
}

func TestCreateElementInvalidTypePanicIncludesActionableGuidance(parseT *testing.T) {
	getPreviousPanicOptions := runtime.CurrentUnhandledPanicLoggingOptions()
	runtime.ConfigureUnhandledPanicLogging(runtime.PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport:           getPreviousPanicOptions.OnReport,
	})
	defer runtime.ConfigureUnhandledPanicLogging(getPreviousPanicOptions)

	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(123))
	if parseErr == nil {
		parseT.Fatal("expected create-element misuse to surface as an SSR error")
	}
	if parseMarkup != "" {
		parseT.Fatalf("expected empty markup for SSR failure, got %q", parseMarkup)
	}
	parseMessage := parseErr.Error()
	if !strings.Contains(parseMessage, "GWC-UI-CREATE-ELEMENT-TYPE") || !strings.Contains(parseMessage, "ACTIONABLE_ERRORS.md#gwc-ui-create-element-type") || !strings.Contains(parseMessage, "where:") || !strings.Contains(parseMessage, "runtime:") {
		parseT.Fatalf("expected actionable create-element error, got %q", parseMessage)
	}
}

func TestRenderUnsupportedOnServerPanicUsesUnifiedContract(parseT *testing.T) {
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected Render to panic on server")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "GWC-UI-UNSUPPORTED-ON-SERVER") || !strings.Contains(parseMessage, "ui.Render") || !strings.Contains(parseMessage, "docs: ACTIONABLE_ERRORS.md#gwc-ui-unsupported-on-server") {
			parseT.Fatalf("expected unified server-only api panic output, got %q", parseMessage)
		}
	}()

	ui.Render(ui.Text("hello"), "#app")
}

func TestAsyncBoundaryAndLazyOnServer(parseT *testing.T) {
	parseFallback := ui.Text("loading")
	parseContent := ui.Text("ready")
	parseContentMarkup, parseContentErr := ui.RenderToString(ui.AsyncBoundary(ui.AsyncBoundaryProps{Content: parseContent}))
	if parseContentErr != nil {
		parseT.Fatalf("unexpected async content render error: %v", parseContentErr)
	}
	if parseContentMarkup != "ready" {
		parseT.Fatalf("expected AsyncBoundary content markup, got %q", parseContentMarkup)
	}
	parsePendingMarkup, parsePendingErr := ui.RenderToString(ui.AsyncBoundary(ui.AsyncBoundaryProps{Pending: true, Fallback: parseFallback}))
	if parsePendingErr != nil {
		parseT.Fatalf("unexpected async pending render error: %v", parsePendingErr)
	}
	if parsePendingMarkup != "loading" {
		parseT.Fatalf("expected AsyncBoundary fallback markup, got %q", parsePendingMarkup)
	}

	parseDone := make(chan struct{})
	parseSuspending := func() ui.Node {
		return ui.Text(ui.Await(ui.SuspenseValue[string]{
			Done:   parseDone,
			Reason: "load greeting",
		}))
	}
	parseSuspendedMarkup, parseSuspendedErr := ui.RenderToString(ui.AsyncBoundary(ui.AsyncBoundaryProps{
		Fallback: parseFallback,
		Content:  ui.CreateElement(parseSuspending),
	}))
	if parseSuspendedErr != nil {
		parseT.Fatalf("unexpected suspended render error: %v", parseSuspendedErr)
	}
	if parseSuspendedMarkup != "loading" {
		parseT.Fatalf("expected Await suspension fallback markup, got %q", parseSuspendedMarkup)
	}

	parseLazy := ui.UseLazyNode(func(parseCtx context.Context) (ui.Node, error) {
		return ui.Text("resolved"), nil
	})
	parseState := parseLazy.Get()
	if !parseState.Ready || parseState.Error != nil || parseState.Node == nil {
		parseT.Fatalf("expected server lazy node to resolve synchronously, got %+v", parseState)
	}

	parseGot3 := ui.Lazy(ui.LazyProps{
		Loader: func(parseCtx2 context.Context) (ui.Node, error) {
			return ui.Text("resolved"), nil
		},
		Fallback: parseFallback,
	})
	parseMarkup, parseErr := ui.RenderToString(parseGot3)
	if parseErr != nil {
		parseT.Fatalf("unexpected async boundary render error: %v", parseErr)
	}
	if parseMarkup != `resolved` {
		parseT.Fatalf("expected lazy server render to resolve content, got %q", parseMarkup)
	}
}

func TestContextProviderWrapperRendersChildrenOnServer(parseT *testing.T) {
	parseTheme := ui.CreateContext("light")
	parseNode := ui.CreateElement(parseTheme.Provider, ui.ContextProviderProps[string]{
		Value: "dark",
		Child: html.P(html.Props{}, html.Text("context shell")),
	})

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("unexpected provider render error: %v", parseErr)
	}
	if parseMarkup != `<p>context shell</p>` {
		parseT.Fatalf("expected provider wrapper to render child subtree, got %q", parseMarkup)
	}
}

func TestPortalRendersChildrenInlineOnServer(parseT *testing.T) {
	parseNode := ui.Portal(ui.PortalProps{
		Target: ui.PortalTarget{Selector: "#overlay-root"},
		Child:  html.P(html.Props{}, html.Text("portal body")),
	})

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("unexpected portal render error: %v", parseErr)
	}
	if parseMarkup != `<p>portal body</p>` {
		parseT.Fatalf("expected server portal fallback to render child inline, got %q", parseMarkup)
	}
}

func TestServerActionResultMapsToFormErrorsOnServer(parseT *testing.T) {
	parseResult := ui.ServerActionResult{
		Outcome: ui.ServerActionOutcomeValidationError,
		Message: "Fix the highlighted fields.",
		Fields:  ui.FieldErrors{"Email": "already used"},
		Redirect: &ui.ServerActionRedirect{
			Location: "/profile",
			Replace:  true,
		},
		Refresh: &ui.ServerActionRefresh{Revalidate: true},
	}
	if !parseResult.HasRedirect() || parseResult.RedirectLocation() != "/profile" {
		parseT.Fatalf("expected redirect metadata, got %+v", parseResult.Redirect)
	}
	if !parseResult.HasRefresh() {
		parseT.Fatal("expected refresh metadata")
	}
	parseProjected := parseResult.FormErrors()
	if parseProjected.FormMessage() != "Fix the highlighted fields." || parseProjected.Fields["Email"] != "already used" {
		parseT.Fatalf("expected typed action result to project into form errors, got %+v", parseProjected)
	}
}

func TestReactiveRegionRendersChildrenOnServer(parseT *testing.T) {
	parseNode := ui.ReactiveRegion(func() ui.Node {
		return html.Span(html.Props{ID: "status"}, html.Text("server region"))
	}, reactiveRegionTestSource{id: "count"})

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("unexpected reactive region render error: %v", parseErr)
	}
	if parseMarkup != `<span id="status">server region</span>` {
		parseT.Fatalf("expected reactive region SSR to render child subtree, got %q", parseMarkup)
	}
}

func TestErrorBoundaryRendersFallbackOnServer(parseT *testing.T) {
	parseBoom := func() ui.Node {
		panic("server boundary")
	}
	parseNode := ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ErrorFallback: func(parseErr2 error, reset func()) ui.Node {
			if parseErr2 == nil || parseErr2.Error() != "server boundary" {
				parseT.Fatalf("unexpected server boundary error: %v", parseErr2)
			}
			return html.P(html.Props{}, html.Text("caught on server"))
		},
		Child: ui.CreateElement(parseBoom),
	})

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("unexpected error boundary render error: %v", parseErr)
	}
	if parseMarkup != `<p>caught on server</p>` {
		parseT.Fatalf("unexpected server boundary fallback markup: %q", parseMarkup)
	}
}

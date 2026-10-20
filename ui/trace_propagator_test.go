//go:build !js || !wasm

package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

type stubPropagator struct{ extracted, injected int }

func (parseP *stubPropagator) Extract(parseCtx context.Context, _ http.Header) context.Context {
	parseP.extracted++
	return ui.WithW3CTraceContext(parseCtx, ui.W3CTraceContext{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736", ParentSpanID: "00f067aa0ba902b7", Flags: "01",
	})
}

func (parseP *stubPropagator) Inject(_ context.Context, parseHeader http.Header) {
	parseP.injected++
	parseHeader.Set("Traceparent", "stub")
}

func TestDefaultPropagatorHonorsSampledFlag(parseT *testing.T) {
	for _, parseCase := range []struct{ flags, want string }{{"01", "-01"}, {"00", "-00"}} {
		parseRec := httptest.NewRecorder()
		parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
		parseReq.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-"+parseCase.flags)
		parseReq.Header.Set("Tracestate", "a=b")
		ui.WrapSSRCorrelation(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(parseRec, parseReq)
		parseTp := parseRec.Header().Get("Traceparent")
		if len(parseTp) < 3 || parseTp[len(parseTp)-3:] != parseCase.want || parseRec.Header().Get("Tracestate") != "a=b" {
			parseT.Fatalf("flags %s: Traceparent=%q Tracestate=%q", parseCase.flags, parseTp, parseRec.Header().Get("Tracestate"))
		}
	}
}

func TestInstalledPropagatorIsUsedAndLegacyFallbackKept(parseT *testing.T) {
	parseStub := &stubPropagator{}
	ui.SetTracePropagator(parseStub)
	parseT.Cleanup(func() { ui.SetTracePropagator(nil) })
	if !ui.IsTracePropagatorInstalled() {
		parseT.Fatal("propagator not reported installed")
	}
	parseRec := httptest.NewRecorder()
	ui.WrapSSRCorrelation(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseHeader := http.Header{}
		ui.InjectTraceContext(parseR.Context(), parseHeader)
		if parseHeader.Get("Traceparent") != "stub" {
			parseT.Error("InjectTraceContext did not call the propagator")
		}
	})).ServeHTTP(parseRec, httptest.NewRequest(http.MethodGet, "/", nil))
	if parseRec.Header().Get("X-Correlation-ID") != "4bf92f3577b34da6a3ce929d0e0e4736" || parseStub.extracted != 1 {
		parseT.Fatalf("extract not honored: %v extracted=%d", parseRec.Header(), parseStub.extracted)
	}

	ui.SetTracePropagator(nil)
	parseRec = httptest.NewRecorder()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Request-ID", "0123456789abcdef")
	ui.WrapSSRCorrelation(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(parseRec, parseReq)
	if parseRec.Header().Get("X-Correlation-ID") != "00000000000000000123456789abcdef" {
		parseT.Fatalf("legacy fallback broken: %v", parseRec.Header())
	}
	parseInject := http.Header{}
	ui.InjectTraceContext(context.Background(), parseInject)
	if len(parseInject) != 0 {
		parseT.Fatal("InjectTraceContext wrote headers with no propagator installed")
	}
}

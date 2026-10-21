//go:build !js || !wasm

package otel

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/monstercameron/GoWebComponents/v7/internal/telemetryredaction"
	"github.com/monstercameron/GoWebComponents/v7/telemetry"
	gwctrace "github.com/monstercameron/GoWebComponents/v7/trace"
)

// getStubAttrs flattens a recorded span's attributes to strings.
func getStubAttrs(parseStub tracetest.SpanStub) map[string]string {
	parseOut := map[string]string{}
	for _, parseKV := range parseStub.Attributes {
		parseOut[string(parseKV.Key)] = parseKV.Value.Emit()
	}
	return parseOut
}

// getStubByName returns the single exported span with parseName.
func getStubByName(parseT *testing.T, parseStubs tracetest.SpanStubs, parseName string) tracetest.SpanStub {
	parseT.Helper()
	for _, parseStub := range parseStubs {
		if parseStub.Name == parseName {
			return parseStub
		}
	}
	parseT.Fatalf("no exported span %q among %d", parseName, len(parseStubs))
	return tracetest.SpanStub{}
}

func TestExportRUMEventsGoldenErrorAttributes(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{})
	parseExporter := tracetest.NewInMemoryExporter()
	parseBridge := NewBridge(Options{TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSyncer(parseExporter))})
	parseWhen := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	parseEvents := []telemetry.RUMEvent{
		{Name: "render.fail", Type: "error", Timestamp: parseWhen, TraceID: "corr-a", Attributes: map[string]string{
			"exception.type": "*errcode.Error", "exception.message": "[GWC-FRAMEWORK-USAGE] misuse: x",
			"gwc.component_stack": "  App > Page > Row \n"}},
		{Name: "render.explicit", Type: "error", Timestamp: parseWhen, TraceID: "corr-b", Attributes: map[string]string{
			"exception.message": "[GWC-OTHER] ignored", "gwc.error.code": "GWC-EXPLICIT"}},
		{Name: "render.ok", Type: "profile", Timestamp: parseWhen, TraceID: "corr-c", Attributes: map[string]string{
			"gwc.component_stack": "   "}},
	}
	if parseErr := parseBridge.ExportRUMEvents(context.Background(), parseEvents); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseStubs := parseExporter.GetSpans()

	parseFail := getStubByName(parseT, parseStubs, "render.fail")
	parseWant := map[string]string{
		"exception.type": "*errcode.Error", "exception.message": "[GWC-FRAMEWORK-USAGE] misuse: x",
		"gwc.error.code": "GWC-FRAMEWORK-USAGE", "gwc.component_stack": "App > Page > Row",
	}
	if parseGot := getStubAttrs(parseFail); !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("render.fail attrs:\n got %v\nwant %v", parseGot, parseWant)
	}
	if parseFail.Status.Code != codes.Error {
		parseT.Fatalf("render.fail status = %v", parseFail.Status)
	}
	if parseGot := getStubAttrs(getStubByName(parseT, parseStubs, "render.explicit"))["gwc.error.code"]; parseGot != "GWC-EXPLICIT" {
		parseT.Fatalf("explicit code overridden: %q", parseGot)
	}
	if parseGot := getStubAttrs(getStubByName(parseT, parseStubs, "render.ok")); len(parseGot) != 0 {
		parseT.Fatalf("blank stack exported: %v", parseGot)
	}
}

func TestExportRecorderSpansGoldenErrorCode(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{})
	parseExporter := tracetest.NewInMemoryExporter()
	parseBridge := NewBridge(Options{TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSyncer(parseExporter))})
	parseRecorder, parseErr := gwctrace.NewRecorder("main", 16)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseRoot, _ := parseRecorder.Start("root", "corr", "", 1000)
	parseKid, _ := parseRecorder.Start("kid", "corr", parseRoot, 2000)
	_ = parseRecorder.End(parseKid, 3000, "[GWC-FRAMEWORK-USAGE] misuse")
	_ = parseRecorder.End(parseRoot, 5000, "")
	parseOrigin := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	if parseErr := parseBridge.ExportRecorderSpans(context.Background(), parseRecorder, parseOrigin); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseStubs := parseExporter.GetSpans()
	parseCorr := ResolveTraceID("corr", 0, "")

	parseKidStub := getStubByName(parseT, parseStubs, "kid")
	parseWant := map[string]string{
		"gwc.thread": "main", "gwc.ssr.correlation_id": "corr", "gwc.error.code": "GWC-FRAMEWORK-USAGE",
	}
	if parseGot := getStubAttrs(parseKidStub); !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("kid attrs:\n got %v\nwant %v", parseGot, parseWant)
	}
	if parseKidStub.Status.Code != codes.Error || parseKidStub.SpanContext.TraceID().String() != parseCorr {
		parseT.Fatalf("kid status=%v trace=%s want %s", parseKidStub.Status, parseKidStub.SpanContext.TraceID(), parseCorr)
	}
	parseRootAttrs := getStubAttrs(getStubByName(parseT, parseStubs, "root"))
	if _, parseHas := parseRootAttrs["gwc.error.code"]; parseHas {
		parseT.Fatalf("root has an error code: %v", parseRootAttrs)
	}
	if !parseKidStub.StartTime.Equal(parseOrigin.Add(2000)) || !parseKidStub.EndTime.Equal(parseOrigin.Add(3000)) {
		parseT.Fatalf("kid times %v %v", parseKidStub.StartTime, parseKidStub.EndTime)
	}
}

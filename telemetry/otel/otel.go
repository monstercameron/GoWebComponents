//go:build !js || !wasm

// Package otel bridges GoWebComponents correlation and telemetry to OpenTelemetry.
//
// It is the only GWC package allowed to import go.opentelemetry.io; ui, serverfn,
// fetch, logging, telemetry and the runtime stay OpenTelemetry-free and expose
// small hooks (ui.SetTracePropagator, serverfn.SetTracePropagator and
// serverfn.SetServerSpanHook) that Install fills in. It is native-only: the
// browser build never links it.
package otel

import (
	"context"
	"sort"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/monstercameron/GoWebComponents/v6/internal/diagnostics"
	"github.com/monstercameron/GoWebComponents/v6/internal/telemetryredaction"
	"github.com/monstercameron/GoWebComponents/v6/serverfn"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ScopeName is the instrumentation scope reported on every span this package starts.
const ScopeName = "github.com/monstercameron/GoWebComponents/v6/telemetry/otel"

// Options configures a Bridge.
type Options struct {
	// TracerProvider creates spans. Defaults to otel.GetTracerProvider().
	TracerProvider trace.TracerProvider
	// Propagator extracts and injects trace context. Defaults to the global
	// propagator, or W3C Trace Context when the global one has no fields.
	Propagator propagation.TextMapPropagator
	// MeterProvider creates the gwc.* instruments. Defaults to otel.GetMeterProvider().
	MeterProvider otelmetric.MeterProvider
}

// Bridge holds the tracer and propagator used by the middleware, the server
// function hook and the replay exporters.
type Bridge struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	metrics    *bridgeMetrics
}

// NewBridge builds a Bridge from parseOptions without touching any global hook.
func NewBridge(parseOptions Options) *Bridge {
	parseProvider := parseOptions.TracerProvider
	if parseProvider == nil {
		parseProvider = otel.GetTracerProvider()
	}
	parsePropagator := parseOptions.Propagator
	if parsePropagator == nil {
		parsePropagator = otel.GetTextMapPropagator()
	}
	if len(parsePropagator.Fields()) == 0 {
		parsePropagator = propagation.TraceContext{}
	}
	parseMeterProvider := parseOptions.MeterProvider
	if parseMeterProvider == nil {
		parseMeterProvider = otel.GetMeterProvider()
	}
	// A partly failed instrument set still yields a usable bridge: failed
	// instruments stay no-ops, so the error is not fatal.
	parseMetrics, _ := buildBridgeMetrics(parseMeterProvider.Meter(ScopeName))
	return &Bridge{tracer: parseProvider.Tracer(ScopeName), propagator: parsePropagator, metrics: parseMetrics}
}

// Install builds a Bridge and registers it with ui and serverfn: the propagator
// on both packages and the server function span hook. The returned shutdown
// function removes the hooks again; it ignores its context because removal is
// immediate.
func Install(parseOptions Options) (*Bridge, func(context.Context) error) {
	parseBridge := NewBridge(parseOptions)
	parseAdapter := parseBridge.Propagator()
	ui.SetTracePropagator(parseAdapter)
	serverfn.SetTracePropagator(parseAdapter)
	serverfn.SetServerSpanHook(parseBridge.startServerFnSpan)
	diagnostics.SetPanicObserver(parseBridge.RecordPanic)
	return parseBridge, func(context.Context) error {
		ui.SetTracePropagator(nil)
		serverfn.SetTracePropagator(nil)
		serverfn.SetServerSpanHook(nil)
		diagnostics.SetPanicObserver(nil)
		parseBridge.closeMetrics()
		return nil
	}
}

// Propagator returns the adapter that satisfies ui.TracePropagator and
// serverfn.TracePropagator.
func (parseBridge *Bridge) Propagator() *PropagatorAdapter {
	return &PropagatorAdapter{propagator: parseBridge.propagator}
}

// redactString runs one value through the telemetry redaction policy. The bool is
// false when the policy dropped the field (fail closed).
func redactString(parseField string, parseValue string) (string, bool) {
	parseOut, parseKeep := telemetryredaction.Field("attributes."+parseField, parseField, parseValue)
	if !parseKeep {
		return "", false
	}
	if parseText, parseIsText := parseOut.(string); parseIsText {
		return parseText, true
	}
	return "", false
}

// convertAttributes is the single place string attributes become OpenTelemetry
// attributes: it applies the redaction policy, drops any attribute whose
// redactor fails or yields a non-string, and returns keys sorted for stable output.
func convertAttributes(parseValues map[string]string) []attribute.KeyValue {
	parseKeys := make([]string, 0, len(parseValues))
	for parseKey := range parseValues {
		if parseKey != "" {
			parseKeys = append(parseKeys, parseKey)
		}
	}
	sort.Strings(parseKeys)
	parseAttrs := make([]attribute.KeyValue, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		if parseValue, parseKeep := redactString(parseKey, parseValues[parseKey]); parseKeep {
			parseAttrs = append(parseAttrs, attribute.String(parseKey, parseValue))
		}
	}
	return parseAttrs
}

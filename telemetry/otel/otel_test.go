//go:build !js || !wasm

package otel

import (
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
	"github.com/monstercameron/GoWebComponents/v6/internal/telemetryredaction"
	"github.com/monstercameron/GoWebComponents/v6/serverfn"
	"github.com/monstercameron/GoWebComponents/v6/telemetry"
	gwctrace "github.com/monstercameron/GoWebComponents/v6/trace"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// fakeProvider is an in-memory TracerProvider: the otel sdk module is not a
// dependency of this repository, so spans are recorded by hand.
type fakeProvider struct {
	embedded.TracerProvider
	mu    sync.Mutex
	next  uint64
	spans []*fakeSpan
}

type fakeTracer struct {
	embedded.Tracer
	provider *fakeProvider
}

type fakeEvent struct {
	name  string
	attrs map[string]string
}

type fakeSpan struct {
	embedded.Span
	mu       sync.Mutex
	sc       trace.SpanContext
	parent   trace.SpanContext
	name     string
	kind     trace.SpanKind
	attrs    map[string]string
	events   []fakeEvent
	status   codes.Code
	message  string
	start    time.Time
	end      time.Time
	hasEnded bool
}

func (parseP *fakeProvider) Tracer(string, ...trace.TracerOption) trace.Tracer {
	return &fakeTracer{provider: parseP}
}

func (parseT *fakeTracer) Start(parseCtx context.Context, parseName string, parseOpts ...trace.SpanStartOption) (context.Context, trace.Span) {
	parseCfg := trace.NewSpanStartConfig(parseOpts...)
	parseP := parseT.provider
	parseP.mu.Lock()
	parseP.next++
	parseN := parseP.next
	parseP.mu.Unlock()
	parseParent := trace.SpanContextFromContext(parseCtx)
	parseConfig := trace.SpanContextConfig{TraceFlags: trace.FlagsSampled}
	var parseSID trace.SpanID
	binary.BigEndian.PutUint64(parseSID[:], 0xA000+parseN)
	parseConfig.SpanID = parseSID
	if parseParent.IsValid() {
		parseConfig.TraceID = parseParent.TraceID()
		parseConfig.TraceFlags = parseParent.TraceFlags()
		parseConfig.TraceState = parseParent.TraceState()
	} else {
		parseConfig.TraceID = trace.TraceID{0xFA, 0xCE, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(parseN)}
	}
	parseSpan := &fakeSpan{
		sc: trace.NewSpanContext(parseConfig), parent: parseParent, name: parseName, kind: parseCfg.SpanKind(),
		attrs: map[string]string{}, start: parseCfg.Timestamp(),
	}
	for _, parseKV := range parseCfg.Attributes() {
		parseSpan.attrs[string(parseKV.Key)] = parseKV.Value.Emit()
	}
	parseP.mu.Lock()
	parseP.spans = append(parseP.spans, parseSpan)
	parseP.mu.Unlock()
	return trace.ContextWithSpan(parseCtx, parseSpan), parseSpan
}

func (parseS *fakeSpan) End(parseOpts ...trace.SpanEndOption) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	parseS.hasEnded = true
	parseEndCfg := trace.NewSpanEndConfig(parseOpts...)
	parseS.end = parseEndCfg.Timestamp()
}
func (parseS *fakeSpan) AddEvent(parseName string, parseOpts ...trace.EventOption) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	parseAttrs := map[string]string{}
	parseEvCfg := trace.NewEventConfig(parseOpts...)
	for _, parseKV := range parseEvCfg.Attributes() {
		parseAttrs[string(parseKV.Key)] = parseKV.Value.Emit()
	}
	parseS.events = append(parseS.events, fakeEvent{name: parseName, attrs: parseAttrs})
}
func (parseS *fakeSpan) AddLink(trace.Link)                      {}
func (parseS *fakeSpan) IsRecording() bool                       { return true }
func (parseS *fakeSpan) RecordError(error, ...trace.EventOption) {}
func (parseS *fakeSpan) SpanContext() trace.SpanContext          { return parseS.sc }
func (parseS *fakeSpan) SetName(parseName string)                { parseS.name = parseName }
func (parseS *fakeSpan) TracerProvider() trace.TracerProvider    { return &fakeProvider{} }
func (parseS *fakeSpan) SetStatus(parseCode codes.Code, parseMessage string) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	parseS.status, parseS.message = parseCode, parseMessage
}
func (parseS *fakeSpan) SetAttributes(parseKVs ...attribute.KeyValue) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	for _, parseKV := range parseKVs {
		parseS.attrs[string(parseKV.Key)] = parseKV.Value.Emit()
	}
}

func (parseP *fakeProvider) byName(parseT *testing.T, parseName string) *fakeSpan {
	parseT.Helper()
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	for _, parseS := range parseP.spans {
		if parseS.name == parseName {
			return parseS
		}
	}
	parseT.Fatalf("no span named %q among %d spans", parseName, len(parseP.spans))
	return nil
}

const (
	testTraceID  = "4bf92f3577b34da6a3ce929d0e0e4736"
	testParentID = "00f067aa0ba902b7"
)

// buildTestServer serves an SSR page that calls the echo server function with its
// request context, plus the echo and failing server functions.
func buildTestServer(parseT *testing.T, parseBridge *Bridge) *httptest.Server {
	parseT.Helper()
	parseMux := http.NewServeMux()
	serverfn.Handle(parseMux, "echo", func(parseCtx context.Context, parseReq map[string]string) (map[string]string, error) {
		return parseReq, nil
	})
	serverfn.Handle(parseMux, "boom", func(context.Context, map[string]string) (map[string]string, error) {
		return nil, errcode.New(errcode.FrameworkUsage)
	})
	parseMux.Handle("/page", parseBridge.Middleware(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		RecordSSRObservation(parseR.Context(), ui.SSRObservation{Name: "render", Domain: "ssr", Phase: "complete", Timestamp: time.Now()})
		if _, parseErr := serverfn.Call[map[string]string, map[string]string](parseR.Context(), "echo", map[string]string{"a": "b"}); parseErr != nil {
			parseT.Errorf("echo: %v", parseErr)
		}
		_, _ = serverfn.Call[map[string]string, map[string]string](parseR.Context(), "boom", nil)
		_, _ = parseW.Write([]byte("ok"))
	})))
	parseSrv := httptest.NewServer(parseMux)
	parseT.Cleanup(parseSrv.Close)
	serverfn.Configure(parseSrv.URL)
	parseT.Cleanup(func() { serverfn.Configure("") })
	return parseSrv
}

func getPage(parseT *testing.T, parseURL string, parseTraceparent string) *http.Response {
	parseT.Helper()
	parseReq, _ := http.NewRequest(http.MethodGet, parseURL+"/page", nil)
	if parseTraceparent != "" {
		parseReq.Header.Set("Traceparent", parseTraceparent)
	}
	parseResp, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseResp.Body.Close()
	return parseResp
}

func TestOneTraceAcrossSSRServerFnAndClient(parseT *testing.T) {
	parseProvider := &fakeProvider{}
	parseBridge, parseShutdown := Install(Options{TracerProvider: parseProvider})
	defer parseShutdown(context.Background())
	parseSrv := buildTestServer(parseT, parseBridge)

	parseResp := getPage(parseT, parseSrv.URL, "00-"+testTraceID+"-"+testParentID+"-01")

	parseSSR := parseProvider.byName(parseT, SSRSpanName)
	if parseSSR.sc.TraceID().String() != testTraceID {
		parseT.Fatalf("ssr trace id = %s, want %s", parseSSR.sc.TraceID(), testTraceID)
	}
	if parseSSR.parent.SpanID().String() != testParentID || !parseSSR.parent.IsRemote() {
		parseT.Fatalf("ssr parent = %v, want remote %s", parseSSR.parent, testParentID)
	}
	if parseSSR.kind != trace.SpanKindServer || !parseSSR.hasEnded {
		parseT.Fatalf("ssr span kind=%v ended=%v", parseSSR.kind, parseSSR.hasEnded)
	}
	if parseSSR.attrs["gwc.ssr.event.name"] != "render" {
		parseT.Fatalf("ssr observation attributes missing: %v", parseSSR.attrs)
	}
	parseEcho := parseProvider.byName(parseT, ServerFnSpanPrefix+"echo")
	if parseEcho.sc.TraceID() != parseSSR.sc.TraceID() || parseEcho.parent.SpanID() != parseSSR.sc.SpanID() {
		parseT.Fatalf("echo span not a child of ssr span: trace=%s parent=%s want %s", parseEcho.sc.TraceID(), parseEcho.parent.SpanID(), parseSSR.sc.SpanID())
	}
	if parseEcho.attrs["http.response.status_code"] != "200" || parseEcho.status != codes.Unset {
		parseT.Fatalf("echo span = %v status %v", parseEcho.attrs, parseEcho.status)
	}

	parseBoom := parseProvider.byName(parseT, ServerFnSpanPrefix+"boom")
	if parseBoom.status != codes.Error || len(parseBoom.events) != 1 {
		parseT.Fatalf("boom status=%v events=%v", parseBoom.status, parseBoom.events)
	}
	if parseBoom.events[0].attrs["gwc.error.code"] != string(errcode.FrameworkUsage) {
		parseT.Fatalf("boom event attrs = %v", parseBoom.events[0].attrs)
	}

	parseWant := "00-" + testTraceID + "-" + parseSSR.sc.SpanID().String() + "-01"
	if parseResp.Header.Get("Traceparent") != parseWant {
		parseT.Fatalf("Traceparent = %q, want %q", parseResp.Header.Get("Traceparent"), parseWant)
	}
	if parseResp.Header.Get("X-Correlation-ID") != testTraceID {
		parseT.Fatalf("X-Correlation-ID = %q", parseResp.Header.Get("X-Correlation-ID"))
	}
}

func TestUnsampledFlagIsHonoredAndNewTraceIsStarted(parseT *testing.T) {
	parseProvider := &fakeProvider{}
	parseBridge, parseShutdown := Install(Options{TracerProvider: parseProvider})
	defer parseShutdown(context.Background())
	parseSrv := buildTestServer(parseT, parseBridge)

	parseResp := getPage(parseT, parseSrv.URL, "00-"+testTraceID+"-"+testParentID+"-00")
	if !strings.HasSuffix(parseResp.Header.Get("Traceparent"), "-00") {
		parseT.Fatalf("unsampled flag lost: %q", parseResp.Header.Get("Traceparent"))
	}
	parseRoot := getPage(parseT, parseSrv.URL, "")
	if !strings.HasSuffix(parseRoot.Header.Get("Traceparent"), "-01") || parseRoot.Header.Get("X-Correlation-ID") == "" {
		parseT.Fatalf("root request headers = %v", parseRoot.Header)
	}
}

func TestMiddlewareFallsBackWithoutAValidSpan(parseT *testing.T) {
	parseBridge := NewBridge(Options{TracerProvider: noop.NewTracerProvider()})
	parseRec := httptest.NewRecorder()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Header.Set("X-Correlation-ID", "req-1")
	parseBridge.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(parseRec, parseReq)
	if parseRec.Header().Get("X-Correlation-ID") == "" {
		parseT.Fatal("fallback did not set X-Correlation-ID")
	}
}

func TestExportRUMEventsReplaysTraceParentsAndErrors(parseT *testing.T) {
	parseProvider := &fakeProvider{}
	parseBridge := NewBridge(Options{TracerProvider: parseProvider})
	parseWhen := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	parseEvents := []telemetry.RUMEvent{
		{Name: "child", Type: "profile", Timestamp: parseWhen.Add(time.Millisecond), DurationNs: 500, TraceID: "corr-1", SpanID: "bbbbbbbbbbbbbbbb", ParentSpanID: "aaaaaaaaaaaaaaaa"},
		{Name: "parent", Type: "ssr", Timestamp: parseWhen, DurationNs: 2000000, TraceID: "corr-1", SpanID: "aaaaaaaaaaaaaaaa"},
		{Name: "fetch.api", Type: "fetch", Timestamp: parseWhen, TraceID: "corr-2", Attributes: map[string]string{"exception.type": "E", "exception.message": "bad thing", "gwc.error.code": "X"}},
	}
	if parseErr := parseBridge.ExportRUMEvents(context.Background(), parseEvents); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseParent := parseProvider.byName(parseT, "parent")
	parseChild := parseProvider.byName(parseT, "child")
	if parseParent.sc.TraceID().String() != ResolveTraceID("corr-1", 0, "") || parseChild.sc.TraceID() != parseParent.sc.TraceID() {
		parseT.Fatalf("trace ids: parent %s child %s", parseParent.sc.TraceID(), parseChild.sc.TraceID())
	}
	if parseChild.parent.SpanID() != parseParent.sc.SpanID() {
		parseT.Fatalf("child parent = %s, want %s", parseChild.parent.SpanID(), parseParent.sc.SpanID())
	}
	if parseParent.kind != trace.SpanKindServer || !parseParent.start.Equal(parseWhen) || !parseParent.end.Equal(parseWhen.Add(2*time.Millisecond)) {
		parseT.Fatalf("parent kind/timestamps wrong: %v %v %v", parseParent.kind, parseParent.start, parseParent.end)
	}
	if parseChild.attrs["gwc.event.span_id"] != "bbbbbbbbbbbbbbbb" {
		parseT.Fatalf("child attrs = %v", parseChild.attrs)
	}
	parseFetch := parseProvider.byName(parseT, "fetch.api")
	if parseFetch.kind != trace.SpanKindClient || parseFetch.status != codes.Error || parseFetch.message != "bad thing" {
		parseT.Fatalf("fetch span = kind %v status %v msg %q", parseFetch.kind, parseFetch.status, parseFetch.message)
	}
	if parseFetch.sc.TraceID() == parseParent.sc.TraceID() {
		parseT.Fatal("distinct correlation ids shared a trace")
	}
}

func TestExportRecorderSpansReplaysFinishedSpans(parseT *testing.T) {
	parseRecorder, parseErr := gwctrace.NewRecorder("main", 16)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseRoot, _ := parseRecorder.Start("root", "corr", "", 1000)
	parseKid, _ := parseRecorder.Start("kid", "corr", parseRoot, 2000)
	_ = parseRecorder.End(parseKid, 3000, "kid failed")
	_ = parseRecorder.End(parseRoot, 5000, "")
	_, _ = parseRecorder.Start("open", "corr", "", 6000)

	parseProvider := &fakeProvider{}
	parseOrigin := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	if parseErr := NewBridge(Options{TracerProvider: parseProvider}).ExportRecorderSpans(context.Background(), parseRecorder, parseOrigin); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if len(parseProvider.spans) != 2 {
		parseT.Fatalf("exported %d spans, want 2 (open span skipped)", len(parseProvider.spans))
	}
	parseRootSpan, parseKidSpan := parseProvider.byName(parseT, "root"), parseProvider.byName(parseT, "kid")
	if parseKidSpan.parent.SpanID() != parseRootSpan.sc.SpanID() || parseKidSpan.status != codes.Error {
		parseT.Fatalf("kid parent=%s status=%v", parseKidSpan.parent.SpanID(), parseKidSpan.status)
	}
	if !parseRootSpan.start.Equal(parseOrigin.Add(1000)) || !parseRootSpan.end.Equal(parseOrigin.Add(5000)) {
		parseT.Fatalf("root times %v %v", parseRootSpan.start, parseRootSpan.end)
	}
}

func TestRedactionAppliesToEveryExportedAttribute(parseT *testing.T) {
	telemetryredaction.Configure(telemetryredaction.Policy{
		Fields: []string{"secret", "broken", "exception.message"},
		Redact: func(parseCtx telemetryredaction.Context) (any, error) {
			if parseCtx.Field == "broken" {
				return nil, errors.New("redactor failed")
			}
			return "[x]", nil
		},
	})
	parseT.Cleanup(func() { telemetryredaction.Configure(telemetryredaction.Policy{}) })

	parseProvider := &fakeProvider{}
	parseBridge := NewBridge(Options{TracerProvider: parseProvider})
	_ = parseBridge.ExportRUMEvents(context.Background(), []telemetry.RUMEvent{{
		Name: "e", Type: "error", TraceID: "c",
		Attributes: map[string]string{"secret": "hunter2", "broken": "leak", "keep": "ok", "exception.message": "token abc"},
	}})
	parseSpan := parseProvider.byName(parseT, "e")
	if parseSpan.attrs["secret"] != "[x]" || parseSpan.attrs["keep"] != "ok" {
		parseT.Fatalf("attrs = %v", parseSpan.attrs)
	}
	if _, parseLeaked := parseSpan.attrs["broken"]; parseLeaked {
		parseT.Fatal("attribute with a failing redactor was exported")
	}
	if parseSpan.message != "[x]" {
		parseT.Fatalf("status message not redacted: %q", parseSpan.message)
	}
}
